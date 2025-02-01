package main

import (
	"encoding/json"
	"fmt"
)

var ForceCharging = false
var ForceChargingByVolt = false

var gridConsumption float32 = 0
var inverterGridOutputPower float32 = 0

var totalConsumption float32
var dischargePowerTotal float32 = 0

var batVoltage float32 = 48
var minSoC float64 = 40
var maxSoC float64 = 99
var batSoC float64 = 0

var maxChargeCurrent float32 = 100
var maxDischargeCurrent float32 = 100

func setBatteryOutput() {
	requiredBatteryCurrent := -totalConsumption / batVoltage

	/* Enforce current limits */
	if requiredBatteryCurrent < maxDischargeCurrent {
		requiredBatteryCurrent = maxDischargeCurrent
	}

	if requiredBatteryCurrent > maxChargeCurrent {
		requiredBatteryCurrent = maxChargeCurrent
	}

	/* Enforce SoC limits */
	if (requiredBatteryCurrent < 0 && batSoC <= minSoC) || requiredBatteryCurrent > 0 && batSoC >= maxSoC {
		requiredBatteryCurrent = 0
		fmt.Printf("SoC limit reached: minSoC:%v maxSoC: %v batSoc: %v\n", minSoC, maxSoC, batSoC)
	}

	fmt.Printf("requiredBatteryCurrent: %v\n", requiredBatteryCurrent)
}

type Inverter struct {
	ID          byte
	Name        string
	SolarPower  float32
	ChargePower float32
	ErrorState  ErrorFlag
}

func calcMaxChargePowerBySoc() (maxChargePowerTotal float32) {
	maxChargePowerTotal = 8000
	if batSoC > 60 {
		maxChargePowerTotal = 6000
	}
	if batSoC > 70 {
		maxChargePowerTotal = 5000
	}
	if batSoC > 75 {
		maxChargePowerTotal = 4000
	}
	if batSoC > 80 {
		maxChargePowerTotal = 3500
	}
	if batSoC > 85 {
		maxChargePowerTotal = 3000
	}
	if batSoC > 90 {
		maxChargePowerTotal = 2500
	}
	if batSoC > 95 {
		maxChargePowerTotal = 1500
	}

	if batSoC > 98 {
		maxChargePowerTotal = 500
	}

	return maxChargePowerTotal
}

func calcMaxDischargePowerBySoc() (maxDischargePowerTotal float32) {
	maxDischargePowerTotal = -6000
	if batSoC < 50 {
		maxDischargePowerTotal = -5000
	}
	if batSoC < 40 {
		maxDischargePowerTotal = -4000
	}
	if batSoC < 30 {
		maxDischargePowerTotal = -2500
	}
	if batSoC < 20 {
		maxDischargePowerTotal = -1500
	}
	if batSoC < 10 {
		maxDischargePowerTotal = -1000
	}

	return maxDischargePowerTotal
}

func Compute() {
	var solarPowerTotal float32 = 0

	for _, inverter := range inverters {
		solarPowerTotal += inverter.SolarPower
	}

	fmt.Printf("solarPowerTotal: %v\n", solarPowerTotal)

	//var minVolt float32 = 44 // 43.2V
	//var maxVolt float32 = 57 // 58.4V

	totalConsumption = gridConsumption + inverterGridOutputPower

	fmt.Printf("gridConsumption: %v\n", gridConsumption)
	fmt.Printf("inverterGridOutputPower: %v\n", inverterGridOutputPower)
	fmt.Printf("totalConsumption: %v\n", totalConsumption)

	// Check how much power we are using from / to battery
	var currentChargePower float32

	for i := range inverters {
		currentChargePower += inverters[i].ChargePower
	}

	var requiredPower float32 = 0
	requiredPower = -gridConsumption + currentChargePower

	fmt.Printf("currentChargePower: %v\n", currentChargePower)
	fmt.Printf("requiredPower: %v\n", requiredPower)

	if MinCellVolt < 3.0 && MaxCellVolt < 3.2 {
		ForceChargingByVolt = true
	} else {
		ForceChargingByVolt = false
	}

	if ForceCharging == true {
		fmt.Printf("ForceCharging is enabled!\n")
		requiredPower = 8000
	}

	if ForceChargingByVolt == true {
		fmt.Printf("ForceChargingByVolt is enabled!\n")
		requiredPower = 8000
	}

	if requiredPower > calcMaxChargePowerBySoc() {
		requiredPower = calcMaxChargePowerBySoc()
		fmt.Printf("Limiting ChargePower due to SoC to %v W\n", requiredPower)
	}

	if requiredPower < calcMaxDischargePowerBySoc() {
		requiredPower = calcMaxDischargePowerBySoc()
		fmt.Printf("limiting DischargePower due to SoC to %v\n", requiredPower)
	}

	if requiredPower < 0 {
		fmt.Printf("DISCARGING BATTERY!\n")

		if MinCellVolt < 3.10 {
			fmt.Printf("Min CellVoltage reached, do not discharge!\n")
			for i := range inverters {
				inverters[i].ChargePower = 0
			}
		} else if batSoC < minSoC {
			fmt.Printf("Min SoC reached, do not discharge!\n")
			for i := range inverters {
				inverters[i].ChargePower = 0
			}
		} else {
			// Calculate total deficit
			dischargePowerPerInverter := requiredPower / float32(len(inverters))
			for i := range inverters {
				// Limit max discharge power per inverter
				if dischargePowerPerInverter > 1500 {
					dischargePowerPerInverter = 1500
				}
				inverters[i].ChargePower = dischargePowerPerInverter
			}
		}
	} else {
		fmt.Printf("CARGING BATTERY!\n")

		if batSoC > maxSoC {
			fmt.Printf("Max SoC reached, do not charge!\n")
			for i := range inverters {
				inverters[i].ChargePower = 0
			}
		} else {
			if solarPowerTotal != 0 {
				for i := range inverters {
					proportion := inverters[i].SolarPower / solarPowerTotal
					chargePower := proportion * (requiredPower)

					if chargePower > inverters[i].SolarPower {
						chargePower = inverters[i].SolarPower
					}

					inverters[i].ChargePower = chargePower
					//fmt.Printf("proportion: %v (%v W)\n", proportion, chargePower)
				}
			}
		}
	}

	for _, inverter := range inverters {
		j, _ := json.Marshal(inverter)
		fmt.Printf("OVERVIEW: %v\n", string(j))
	}

	return
}
