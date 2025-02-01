package main

import (
	"encoding/json"
	"fmt"
	"time"
)

var MinCellVolt float64
var MaxCellVolt float64

type State int64

const (
	IDLE State = iota
	CHARGING
	DISCHARGING
)

func (s State) String() string {
	switch s {
	case IDLE:
		return "idle"
	case CHARGING:
		return "charging"
	case DISCHARGING:
		return "discharging"
	default:
		return fmt.Sprintf("unknown state '%d'", s)
	}
}

type EG4 struct {
	ID int `json:"id"`

	BatteryAmps     float64 `json:"batt_a"`
	BatteryVolts    float64 `json:"batt_v"`
	BatterySOC      float64 `json:"batt_soc"`
	BatteryCycles   int64   `json:"batt_cycles"`
	BatterySOH      float64 `json:"battery_soh"`
	BatteryCapacity float64 `json:"battery_capacity"`

	Cell [16]float64 `json:"cells"`

	Temp1   float64 `json:"temp_1"`
	Temp2   float64 `json:"temp_2"`
	Temp3   float64 `json:"temp_3"`
	Temp4   float64 `json:"temp_4"`
	MOSTemp float64 `json:"mos_temp"`
	EnvTemp float64 `json:"env_temp"`

	State   State `json:"state"`
	IsValid bool  `json:"is_valid"`
}

func Parse(b []byte) (EG4, error) {
	//eg4 := &EG4{}
	var eg4 EG4

	eg4.ID = int(b[1])

	g := 4
	l := 5
	gb := byte(0x01)
	lb := byte(0x10)
	if b[g] != gb || b[l] != lb {
		return eg4, fmt.Errorf("expected b[%d] to be %d and b[%d] to be %d and instead they were 0x%02X and 0x%02X", g, gb, l, lb, b[g], b[l])
	}
	eg4.Cell[0] = convertCell(b[6], b[7])
	eg4.Cell[1] = convertCell(b[8], b[9])
	eg4.Cell[2] = convertCell(b[10], b[11])
	eg4.Cell[3] = convertCell(b[12], b[13])
	eg4.Cell[4] = convertCell(b[14], b[15])
	eg4.Cell[5] = convertCell(b[16], b[17])
	eg4.Cell[6] = convertCell(b[18], b[19])
	eg4.Cell[7] = convertCell(b[20], b[21])
	eg4.Cell[8] = convertCell(b[22], b[23])
	eg4.Cell[9] = convertCell(b[24], b[25])
	eg4.Cell[10] = convertCell(b[26], b[27])
	eg4.Cell[11] = convertCell(b[28], b[29])
	eg4.Cell[12] = convertCell(b[30], b[31])
	eg4.Cell[13] = convertCell(b[32], b[33])
	eg4.Cell[14] = convertCell(b[34], b[35])
	eg4.Cell[15] = convertCell(b[36], b[37])

	g = 38
	l = 39
	gb = 0x02
	lb = 0x01
	if b[g] != gb || b[l] != lb {
		return eg4, fmt.Errorf("expected b[%d] to be %d and b[%d] to be %d and instead they were 0x%02X and 0x%02X", g, gb, l, lb, b[g], b[l])
	}
	eg4.BatteryAmps = convertAmps(b[40], b[41])

	g = 42
	l = 43
	gb = 0x03
	lb = 0x01
	if b[g] != gb || b[l] != lb {
		return eg4, fmt.Errorf("expected b[%d] to be %d and b[%d] to be %d and instead they were 0x%02X and 0x%02X", g, gb, l, lb, b[g], b[l])
	}
	eg4.BatterySOC = convertSOC(b[44], b[45])

	g = 50
	l = 51
	gb = 0x05
	lb = 0x06
	if b[g] != gb || b[l] != lb {
		return eg4, fmt.Errorf("expected b[%d] to be %d and b[%d] to be %d and instead they were 0x%02X and 0x%02X", g, gb, l, lb, b[g], b[l])
	}
	eg4.Temp1 = convertTemp(b[53])
	eg4.Temp2 = convertTemp(b[55])
	eg4.Temp3 = convertTemp(b[57])
	eg4.Temp4 = convertTemp(b[59])
	eg4.MOSTemp = convertTemp(b[61])
	eg4.EnvTemp = convertTemp(b[63])

	g = 64
	l = 65
	gb = 0x06
	lb = 0x05
	if b[g] != gb || b[l] != lb {
		return eg4, fmt.Errorf("expected b[%d] to be %d and b[%d] to be %d and instead they were 0x%02X and 0x%02X", g, gb, l, lb, b[g], b[l])
	}
	eg4.State = convertState(b[69])

	g = 76
	l = 77
	gb = 0x07
	lb = 0x01
	if b[g] != gb || b[l] != lb {
		return eg4, fmt.Errorf("expected b[%d] to be %d and b[%d] to be %d and instead they were 0x%02X and 0x%02X", g, gb, l, lb, b[g], b[l])
	}
	eg4.BatteryCycles = convertCycles(b[78], b[79])

	g = 80
	l = 81
	gb = 0x08
	lb = 0x01
	if b[g] != gb || b[l] != lb {
		return eg4, fmt.Errorf("expected b[%d] to be %d and b[%d] to be %d and instead they were 0x%02X and 0x%02X", g, gb, l, lb, b[g], b[l])
	}
	eg4.BatteryVolts = convertVolts(b[82], b[83])

	g = 84
	l = 85
	gb = 0x09
	lb = 0x01
	if b[g] != gb || b[l] != lb {
		return eg4, fmt.Errorf("expected b[%d] to be %d and b[%d] to be %d and instead they were 0x%02X and 0x%02X", g, gb, l, lb, b[g], b[l])
	}
	eg4.BatterySOH = convertSOH(b[86], b[87])

	eg4.BatteryCapacity = convertSOH(b[48], b[49])

	eg4.IsValid = true

	return eg4, nil
}

func convertCell(high, low byte) float64 {
	return float64(uint16(high&0b01111111)<<8|(uint16(low))) / 1000
}

func convertAmps(high, low byte) float64 {
	return (30000 - float64(uint16(high)<<8|(uint16(low)))) / 100
}

func convertSOC(high, low byte) float64 {
	return float64(uint16(high)<<8|(uint16(low))) / 100
}

func convertTemp(low byte) float64 {
	return float64(low) - 50
}

func convertState(by byte) State {
	return State(by)
}

func convertCycles(high, low byte) int64 {
	return int64(uint16(high)<<8 | (uint16(low)))
}

func convertVolts(high, low byte) float64 {
	return float64(uint16(high)<<8|(uint16(low))) / 100
}

func convertSOH(high, low byte) float64 {
	return float64(uint16(high)<<8|(uint16(low))) / 100
}

func calcCheckSum(buf []byte, len byte) byte {
	var i byte
	var num byte
	var num1 byte

	for i = 0; i < len; i++ {
		num = num ^ buf[i]
		num1 += buf[i]
	}

	return (num ^ num1) & 255
}

func validateCheckSum(buf []byte) bool {
	var buffLen byte = byte(len(buf))

	//fmt.Printf("buffLen: %v\n", buffLen)

	if buffLen < 4 {
		return false
	}

	chkSum := calcCheckSum(buf, buffLen-2)
	if chkSum == buf[buffLen-2] {
		return true
	}

	return false
}

var soi byte = 0x7e
var eoi byte = 0x0d
var adr byte = 0x01
var BmsData EG4

func UpdateBmsData() {
	var err error
	txData := []byte{soi, adr, 0x01, 0x00, 0xfe, eoi}
	txData[len(txData)-2] = calcCheckSum(txData, byte(len(txData)-2))

	sendDataToUart(txData, 2000*time.Millisecond, 1)
	rxData := respBuffer.Bytes()

	isOk := validateCheckSum(rxData)

	BmsData.IsValid = false

	if isOk && len(rxData) > 0 {
		BmsData, err = Parse(rxData)

		if err != nil {
			fmt.Printf("Error parse: %v\n", err)
		}

		//fmt.Printf("bmsData: %v\n", BmsData)

		j, _ := json.Marshal(BmsData)
		fmt.Printf("bmsData: %v\n", string(j))

		//tmpStr = fmt.Sprintf("%02X", data)

		//fmt.Printf("\n SOC: %v\n", BmsData.BatterySOC)
	} else {
		fmt.Printf("BMS communication error: %v\n", err)
	}

	GetMinMaxBSMVolts()
}

func GetMinMaxBSMVolts() {
	if BmsData.IsValid == true {
		batSoC = BmsData.BatterySOC

		// Find max and min cell
		MinCellVolt = BmsData.Cell[0]
		MaxCellVolt = BmsData.Cell[0]

		for _, cellVolt := range BmsData.Cell {
			if cellVolt <= MinCellVolt {
				MinCellVolt = cellVolt
			}

			if cellVolt >= MaxCellVolt {
				MaxCellVolt = cellVolt
			}
		}
	}
}
