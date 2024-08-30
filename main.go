package main

import (
	"encoding/json"
	"fmt"
	"github.com/goburrow/modbus"
	"time"
)

var minVolt float32 = 44 // 43.2V
var maxVolt float32 = 57 // 58.4V
var minCellVolt float64
var maxCellVolt float64

func setComParameters(handler *modbus.RTUClientHandler) {
	handler.BaudRate = 115200
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 2
	handler.SlaveId = 0x80
	handler.Timeout = 100 * time.Millisecond
	handler.RS485.Enabled = true
	//handler.Config.RS485.DelayRtsBeforeSend = 200
	//handler.Config.RS485.DelayRtsAfterSend = 300
	//handler.Config.RS485.DelayRtsBeforeSend = 100 * time.Millisecond
	//handler.Config.RS485.DelayRtsAfterSend = 100 * time.Millisecond
	handler.RS485.DelayRtsBeforeSend = 1 * time.Millisecond
	handler.RS485.DelayRtsAfterSend = 0 * time.Millisecond
	handler.RS485.RxDuringTx = false
}

var inverters []Inverter

func main() {
	var err error
	var invReply InverterReply
	var janizzaData JanizzaData

	inverters = []Inverter{
		{0x80, "Inverter1", 0, 0, 0},
		{0x81, "Inverter2", 0, 0, 0},
		{0x82, "Inverter3", 0, 0, 0},
		{0x83, "Inverter4", 0, 0, 0},
	}

	//comPort := "COM5"

	comPort := "/dev/ttySTM1"
	handler := modbus.NewRTUClientHandler(comPort)

	var address uint16
	var results []byte

	address = 0

	setComParameters(handler)
	//handler.Logger = log.New(os.Stdout, "test: ", log.LstdFlags)

	err = handler.Connect()

	if err != nil {
		fmt.Printf("Connect error: %v\n", err)
		return
	}

	fmt.Printf("Opened Com Port: %v Speed: %v\n", comPort, handler.BaudRate)
	client := modbus.NewClient(handler)

	UartInitBasen()

	// Startup
	for _, inverter := range inverters {
		handler.SlaveId = inverter.ID
		//WriteData(client, handler.SlaveId, 47, 10) // 0x2F
		//WriteData(client, handler.SlaveId, 48, 20)
		//WriteData(client, handler.SlaveId, 49, 50)
		//WriteData(client, handler.SlaveId, 50, 1380)

		WriteData(client, handler.SlaveId, 0x2F, 0x0A) // 0x2F
		WriteData(client, handler.SlaveId, 0x30, 0x0014)
		WriteData(client, handler.SlaveId, 0x31, 0x0032)
		WriteData(client, handler.SlaveId, 0x32, 0x0564)
	}

	for {
		fmt.Printf("---\n")
		UpdateBmsData()

		if BmsData.IsValid == true {
			batSoC = BmsData.BatterySOC

			// Find max and min cell
			minCellVolt = BmsData.Cell[0]
			maxCellVolt = BmsData.Cell[0]

			for _, cellVolt := range BmsData.Cell {
				if cellVolt <= minCellVolt {
					minCellVolt = cellVolt
				}

				if cellVolt >= maxCellVolt {
					maxCellVolt = cellVolt
				}
			}
		}

		fmt.Printf("BatSoC: %v, min: %vV, max: %vV\n", batSoC, minCellVolt, maxCellVolt)

		// Get Janizza Data
		handler.SlaveId = 0x01
		err = JanizzaGetRealPower(client, &janizzaData)
		if err != nil {
			fmt.Printf("JanizzaGetRealPower error: %v\n", err)
			continue
		}
		err = JanizzaGetApparentPower(client, &janizzaData)
		if err != nil {
			fmt.Printf("JanizzaGetApparentPower error: %v\n", err)
			continue
		}
		err = JanizzaGetVoltage(client, &janizzaData)
		if err != nil {
			fmt.Printf("JanizzaGetVoltage error: %v\n", err)
			continue
		}

		// Get Inverter Data
		inverterGridOutputPower = 0

		for i := range inverters {
			handler.SlaveId = inverters[i].ID
			results, err = client.ReadHoldingRegisters(address, 58)

			if err != nil {
				fmt.Printf("ReadHoldingRegisters ID: %v Address: %v Error: %v\n", inverters[i].ID, address, err)
			} else {

				if len(results) != 116 {
					fmt.Printf("ReadHoldingRegisters ID: %v Address: %v Length: %v!=116\n", inverters[i].ID, address, len(results))
					continue
				}

				invReply = ParseInverterReadOut(results)
				j, _ := json.Marshal(invReply)
				fmt.Printf("invReply (%v): %v\n", inverters[i].ID, string(j))

				inverters[i].SolarPower = invReply.PSolar
				inverterGridOutputPower += invReply.PGrid

				inverters[i].ErrorState = ErrorFlag(invReply.ErrorState)
				CheckErrors(inverters[i].ErrorState)
			}
		}

		gridConsumption = janizzaData.PSum

		Compute()

		for _, inverter := range inverters {
			handler.SlaveId = inverter.ID
			WriteData(client, inverter.ID, 0x04, 0x1388)

			SetBatteryPower(client, inverter.ID, inverter.ChargePower)

			WriteData(client, inverter.ID, 0x2C, 0x0000) // Netzleistung???
			//WriteData(client, id, 0x2F, 0x000A)
		}

		/*
			// Set Inverter Data
			for _, id := range ids {
				WriteData(client, id, 0x04, 0x1388)
				//WriteData(client, 0x16, 0x0000) // BAT_I
				//WriteData(client, id, 0x16, uint16(testVal)) // BAT_I
				//SetBatteryPower(client, id, -1500)
				SetBatteryPower(client, id, 0)
				//SetBatteryPower(client, id, -janizzaData.PSum)

				//SetPowerLimit(client, id, 100)
				//WriteData(client, id, 0x16, 0x0000) // BAT_I
				WriteData(client, id, 0x2C, 0x0000) // Netzleistung???
				//WriteData(client, id, 0x2F, 0x000A) // ???

				time.Sleep(100 * time.Millisecond)
			}
		*/
		time.Sleep(5000 * time.Millisecond)
	}

	// Shutdown
	/*
		for _, id := range ids {
			handler.SlaveId = id
			WriteData(client, handler.SlaveId, 0x16, 0x0000)
			WriteData(client, handler.SlaveId, 0x2C, 0x0000)
			WriteData(client, handler.SlaveId, 0x2F, 0x000A) // ???
		}
	*/

	handler.Close()
}
