package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/goburrow/modbus"
	"math"
)

var transformerRatio float32 = 12.0

type JanizzaData struct {
	UL1  float32
	UL2  float32
	UL3  float32
	PL1  float32
	PL2  float32
	PL3  float32
	PSum float32
	QL1  float32
	QL2  float32
	QL3  float32
	QSum float32
}

func JanizzaGetRealPower(client modbus.Client, janizzaData *JanizzaData) error {
	// 1020 in old controller
	results, err := client.ReadHoldingRegisters(19020, 8)

	if err != nil {
		return err
	}

	if len(results) != 16 {
		return errors.New("wrong modbus reply")
	}

	l1Temp := binary.BigEndian.Uint32(results[0:4])
	l2Temp := binary.BigEndian.Uint32(results[4:8])
	l3Temp := binary.BigEndian.Uint32(results[8:12])
	SumTemp := binary.BigEndian.Uint32(results[12:16])

	janizzaData.PL1 = math.Float32frombits(l1Temp) * transformerRatio
	janizzaData.PL2 = math.Float32frombits(l2Temp) * transformerRatio
	janizzaData.PL3 = math.Float32frombits(l3Temp) * transformerRatio
	janizzaData.PSum = math.Float32frombits(SumTemp) * transformerRatio
	fmt.Printf("RealPower L1: %v L2: %v L3: %v Sum: %v\n", janizzaData.PL1, janizzaData.PL2, janizzaData.PL3, janizzaData.PSum)

	return nil
}

func JanizzaGetApparentPower(client modbus.Client, janizzaData *JanizzaData) error {
	// 1028 in old controller
	results, err := client.ReadHoldingRegisters(19028, 8)

	if err != nil {
		return err
	}

	if len(results) != 16 {
		return errors.New("wrong modbus reply")
	}

	l1Temp := binary.BigEndian.Uint32(results[0:4])
	l2Temp := binary.BigEndian.Uint32(results[4:8])
	l3Temp := binary.BigEndian.Uint32(results[8:12])
	SumTemp := binary.BigEndian.Uint32(results[12:16])

	janizzaData.QL1 = math.Float32frombits(l1Temp) * transformerRatio
	janizzaData.QL2 = math.Float32frombits(l2Temp) * transformerRatio
	janizzaData.QL3 = math.Float32frombits(l3Temp) * transformerRatio
	janizzaData.QSum = math.Float32frombits(SumTemp) * transformerRatio
	fmt.Printf("ApparentPower L1: %v L2: %v L3: %v Sum: %v\n", janizzaData.QL1, janizzaData.QL2, janizzaData.QL3, janizzaData.QSum)

	return nil
}

func JanizzaGetVoltage(client modbus.Client, janizzaData *JanizzaData) error {
	// 1000 in old controller
	results, err := client.ReadHoldingRegisters(19000, 6)

	if err != nil {
		return err
	}

	if len(results) != 12 {
		return errors.New("wrong modbus reply")
	}

	l1Temp := binary.BigEndian.Uint32(results[0:4])
	l2Temp := binary.BigEndian.Uint32(results[4:8])
	l3Temp := binary.BigEndian.Uint32(results[8:12])

	janizzaData.UL1 = math.Float32frombits(l1Temp)
	janizzaData.UL2 = math.Float32frombits(l2Temp)
	janizzaData.UL3 = math.Float32frombits(l3Temp)
	fmt.Printf("Voltage L1: %v L2: %v L3: %v\n", janizzaData.UL1, janizzaData.UL2, janizzaData.UL3)

	return nil
}
