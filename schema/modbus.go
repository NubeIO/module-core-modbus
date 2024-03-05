package schema

type DataType struct {
	Type     string   `json:"type" default:"string"`
	Title    string   `json:"title" default:"Data Type"`
	Options  []string `json:"enum" default:"[\"digital\",\"uint16\",\"int16\",\"uint32\",\"int32\",\"uint64\",\"int64\",\"float32\",\"float64\",\"mod10-u32\"]"`
	EnumName []string `json:"enumNames" default:"[\"digital\",\"uint16\",\"int16\",\"uint32\",\"int32\",\"uint64\",\"int64\",\"float32\",\"float64\",\"mod10-u32\"]"`
	Default  string   `json:"default" default:"uint16"`
	ReadOnly bool     `json:"readOnly" default:"false"`
}

type ObjectEncoding struct {
	Type     string   `json:"type" default:"string"`
	Title    string   `json:"title" default:"Object Encoding (Endianness)"`
	Options  []string `json:"enum" default:"[\"beb_bew\",\"leb_bew\",\"beb_lew\",\"leb_lew\",]"`
	EnumName []string `json:"enumNames" default:"[\"Standard/Network Order (ABCD)\",\"Byte Swap (BADC)\",\"Word Swap (CDAB)\",\"Byte Swap + Word Swap (DCBA)\"]"`
	Default  string   `json:"default" default:"beb_lew"`
	ReadOnly bool     `json:"readOnly" default:"false"`
}

type ObjectTypeModbus struct {
	Type     string   `json:"type" default:"string"`
	Title    string   `json:"title" default:"Object Type"`
	Options  []string `json:"enum" default:"[\"coil\",\"discrete_input\",\"input_register\",\"holding_register\"]"`
	EnumName []string `json:"enumNames" default:"[\"Coil\",\"Discrete Input\",\"Input Register\",\"Holding Register\"]"`
	Default  string   `json:"default" default:"coil"`
	ReadOnly bool     `json:"readOnly" default:"false"`
}

type SerialPortModbus struct {
	Type     string   `json:"type" default:"string"`
	Title    string   `json:"title" default:"Serial Port"`
	Options  []string `json:"enum" default:"[\"/dev/ttyAMA0\",\"/dev/ttyRS485-1\",\"/dev/ttyRS485-2\",\"/data/socat/loRa1\",\"/dev/ttyUSB0\",\"/dev/ttyUSB1\",\"/dev/ttyUSB2\",\"/dev/ttyUSB3\",\"/dev/ttyUSB4\",\"/data/socat/serialBridge1\",\"/dev/ttyACM0\"]"`
	EnumName []string `json:"enumNames" default:"[\"/dev/ttyAMA0\",\"/dev/ttyRS485-1\",\"/dev/ttyRS485-2\",\"/data/socat/loRa1\",\"/dev/ttyUSB0\",\"/dev/ttyUSB1\",\"/dev/ttyUSB2\",\"/dev/ttyUSB3\",\"/dev/ttyUSB4\",\"/data/socat/serialBridge1\",\"/dev/ttyACM0\"]"`
	Default  string   `json:"default" default:"/dev/ttyRS485-2"`
	ReadOnly bool     `json:"readOnly" default:"false"`
}
