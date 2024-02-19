import requests
import time
import sys

if len(sys.argv) != 2:
    print('please provide IP and port (i.e. 192.168.15.10:1660)')
    exit(1)

base_api_url = f"http://{sys.argv[1]}"
sleep_time = 5
expected_count = 0
wrong_count = 0
incorrect_counter = 0
refresh_counter = 0


def getPointCount():
    resp = requests.get(
        f'{base_api_url}/api/networks?with_devices=true&with_points=true')
    if resp.status_code != 200:
        print(f'ERROR getting points: {resp.status_code} {resp.text}')
        exit(1)
    networks = resp.json()

    global expected_count
    expected_count_orig = expected_count
    expected_count = 0
    for net in networks:
        if net['uuid'] not in network_uuids:
            continue
        if not net['enable']:
            continue
        for dev in net['devices']:
            if not dev['enable']:
                continue
            for point in dev['points']:
                if not point['enable']:
                    continue
                expected_count += 1
    if expected_count != expected_count_orig:
        global wrong_count
        global incorrect_counter
        wrong_count = 0
        incorrect_counter = 0


def getNetPollStats(net_name):
    resp = requests.get(
        f"{base_api_url}/api/modules/module-core-modbus/api/polling/stats/network/name/{net_name}")
    if resp.status_code != 200:
        print(f"Request failed with status code: {resp.status_code}")
        exit(1)

    return resp.json()


resp = requests.get(
    f"{base_api_url}/api/networks")
if resp.status_code != 200:
    print(f"Request failed with status code: {resp.status_code}")
    exit(1)
networks_all = resp.json()
networks = []
network_uuids = []
for net in networks_all:
    if net['plugin_name'] == "module-core-modbus":
        networks.append(net)
        network_uuids.append(net['uuid'])

while True:
    if refresh_counter % 5 == 0:
        getPointCount()
    refresh_counter += 1

    len_poll = 0
    len_stdby = 0
    len_out = 0
    total_count = 0
    lockouts = False
    for net in networks:
        data = getNetPollStats(net['name'])
        if not data['enable']:
            continue
        lockouts |= \
            data["asap_priority_lockup_alert"] or \
            data["high_priority_lockup_alert"] or \
            data["normal_priority_lockup_alert"] or \
            data["low_priority_lockup_alert"]

        len_poll += data["total_poll_queue_length"]
        len_stdby += data["total_standby_points_length"]
        len_out += data["total_points_out_for_polling"]
    total_count += len_poll + len_stdby + len_out

    if total_count != expected_count:
        wrong_count = total_count
        incorrect_counter += 1
        getPointCount()
    elif incorrect_counter:
        incorrect_counter = 0

    print(f'Lockouts: {lockouts}')
    print(f'expec points: {expected_count}')
    print(f'total points: {total_count}')
    print(f'    out:   {len_out}')
    print(f'    poll:  {len_poll}')
    print(f'    stdby: {len_stdby}')

    if incorrect_counter > 1:
        print(f'INCORRECT COUNT: {wrong_count}')

    print("")
    time.sleep(sleep_time)
