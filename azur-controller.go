package main

import (
	"encoding/binary"
	"fmt"
	"github.com/goburrow/modbus"
)

type InverterReply struct {
	val0             uint16
	softwareVersion1 uint16
	val2             uint16
	val3             uint16
	val4             uint16
	val5             uint16
	val6             uint16
	onTime1          uint32 // 7 + 8
	onTime2          uint32 // 9 + 10
	val23            uint16
	val27            int16
	softwareVersion2 uint16
	UBat             float32
	IBat             float32
	USolar           float32
	ISolar           float32
	PSolar           float32
	Temperature      float32
	UGrid            float32
	IGrid            float32
	PGrid            float32
	SGrid            float32

	SPSStatus   uint16
	ErrorState  uint16
	ErrorState2 uint16
	fGrid       uint16
}

var retryCount = 3

func GetModbusRegister(modbusData []byte, register int) uint16 {
	register += 0
	return uint16(modbusData[(register*2)+1]) | uint16(modbusData[register*2])<<8
	//return uint16(modbusData[(register*2)]) | uint16(modbusData[((register*2)+1)])<<8
}

func GetModbusRegisterUint32(modbusData []byte, register int) uint32 {
	return binary.BigEndian.Uint32(modbusData[register*2 : register*2+4])
	//register += 0
	//return uint32(modbusData[(register*2)+1]) | uint32(modbusData[register*2])<<8
	//return uint16(modbusData[(register*2)]) | uint16(modbusData[((register*2)+1)])<<8
}

func ParseInverterReadOut(modbusData []byte) InverterReply {
	var invReply InverterReply
	//invReply.UBat = GetModbusRegister(modbusData, 1)
	invReply.val0 = GetModbusRegister(modbusData, 0)
	invReply.softwareVersion1 = GetModbusRegister(modbusData, 1)
	invReply.val2 = GetModbusRegister(modbusData, 2)
	invReply.val3 = GetModbusRegister(modbusData, 3)
	invReply.val4 = GetModbusRegister(modbusData, 4)
	invReply.val5 = GetModbusRegister(modbusData, 5)
	invReply.val6 = GetModbusRegister(modbusData, 6)
	invReply.onTime1 = GetModbusRegisterUint32(modbusData, 7)
	invReply.onTime2 = GetModbusRegisterUint32(modbusData, 9)
	invReply.softwareVersion2 = GetModbusRegister(modbusData, 19)
	invReply.val23 = GetModbusRegister(modbusData, 23)
	invReply.val27 = int16(GetModbusRegister(modbusData, 27))
	invReply.UBat = float32(GetModbusRegister(modbusData, 25)) / 10.0
	invReply.IBat = float32(int16(GetModbusRegister(modbusData, 26))) / 10.0
	invReply.USolar = float32(GetModbusRegister(modbusData, 28)) / 10.0
	invReply.ISolar = float32(GetModbusRegister(modbusData, 29)) / 10.0
	invReply.PSolar = float32(GetModbusRegister(modbusData, 30))
	invReply.UGrid = float32(GetModbusRegister(modbusData, 31)) / 10.0
	invReply.IGrid = float32(GetModbusRegister(modbusData, 32)) / 10.0
	invReply.PGrid = float32(GetModbusRegister(modbusData, 33))
	invReply.Temperature = float32(GetModbusRegister(modbusData, 36)) / 10.0
	invReply.SPSStatus = GetModbusRegister(modbusData, 37)
	invReply.ErrorState = GetModbusRegister(modbusData, 38)
	invReply.ErrorState2 = GetModbusRegister(modbusData, 39)
	invReply.SGrid = float32(GetModbusRegister(modbusData, 40))
	invReply.fGrid = GetModbusRegister(modbusData, 42)

	return invReply
}

func WriteData(client modbus.Client, id byte, address uint16, value uint16) error {
	var err error
	var results []byte

	for i := 0; i < retryCount; i++ {
		results, err = client.WriteSingleRegister(address, value)

		if err != nil {
			fmt.Printf("ERROR: WriteSingleRegister (ID %v): %v %v %v\n", id, address, results, err)
		} else {
			//fmt.Printf("WriteSingleRegister (ID %v): %v %v\n", id, address, len(results))
			return nil
		}
	}

	return err
}

func SetBatteryPower(client modbus.Client, id byte, powerInW float32) {
	var power int16
	power = int16(powerInW)
	WriteData(client, id, 0x16, uint16(power)) // BAT_I
	//fmt.Printf("SetBatteryPower (ID %v): %v %v %v\n", id, powerInW, power, uint16(power))
}

func SetPowerLimit(client modbus.Client, id byte, powerLimit float32) {
	var current int16
	current = int16(powerLimit * 10.0)
	WriteData(client, id, 0x2C, uint16(current)) // PowerLimit
}
