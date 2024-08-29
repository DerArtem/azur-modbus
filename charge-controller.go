package main

import (
	"encoding/json"
	"fmt"
)

var forceCharging = false

var gridConsumption float32 = 0
var inverterGridOutputPower float32 = 0

var totalConsumption float32
var dischargePowerTotal float32 = 0

var batVoltage float32 = 48
var minSoC float64 = 10
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
	ID             byte
	Name           string
	SolarPower     float32
	ChargePower    float32
	DischargePower float32
	ErrorState     ErrorFlag
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
	maxDischargePowerTotal = 6000
	if batSoC < 50 {
		maxDischargePowerTotal = 5000
	}
	if batSoC < 40 {
		maxDischargePowerTotal = 4000
	}
	if batSoC < 30 {
		maxDischargePowerTotal = 2500
	}
	if batSoC < 20 {
		maxDischargePowerTotal = 1500
	}
	if batSoC < 10 {
		maxDischargePowerTotal = 1000
	}

	return maxDischargePowerTotal
}

func Compute() {
	var solarPowerTotal float32
	//var minVolt float32 = 44 // 43.2V
	//var maxVolt float32 = 57 // 58.4V

	totalConsumption = gridConsumption + inverterGridOutputPower
	//if totalConsumption
	//requiredBatteryPower := gridConsumption

	fmt.Printf("gridConsumption: %v\n", gridConsumption)
	fmt.Printf("inverterGridOutputPower: %v\n", inverterGridOutputPower)
	fmt.Printf("totalConsumption: %v\n", totalConsumption)

	//fmt.Printf("requiredBatteryPower: %v\n", requiredBatteryPower)

	solarPowerTotal = 0

	for _, inverter := range inverters {
		solarPowerTotal += inverter.SolarPower
	}

	fmt.Printf("solarPowerTotal: %v\n", solarPowerTotal)

	if totalConsumption < solarPowerTotal || forceCharging == true {
		surplusPower := -gridConsumption

		if forceCharging == true {
			fmt.Printf("ForceCharging is enabled!\n")
			surplusPower = solarPowerTotal
		}

		if surplusPower > 0.1 {
			// Charge the battery:
			fmt.Printf("Surplus energy availible, charging battery: %v W\n", surplusPower)

			if surplusPower > calcMaxChargePowerBySoc() {
				surplusPower = calcMaxChargePowerBySoc()
				fmt.Printf("Limiting ChargePower due to SoC to %v W\n", surplusPower)
			}

			dischargePowerTotal = 0
			//surplusPower := gridConsumption - solarPowerTotal

			if solarPowerTotal != 0 {
				for i := range inverters {
					proportion := inverters[i].SolarPower / solarPowerTotal
					chargePower := proportion * (surplusPower + inverterGridOutputPower)

					if chargePower > inverters[i].SolarPower {
						chargePower = inverters[i].SolarPower
					}

					inverters[i].ChargePower = chargePower
					//fmt.Printf("proportion: %v (%v W)\n", proportion, chargePower)
				}
			}

			if batSoC > maxSoC {
				fmt.Printf("Max SoC reached, do not charge!\n")
				for i := range inverters {
					inverters[i].ChargePower = 0
				}
			}
		} else {
			fmt.Printf("Surplus energy availible, but negative: %v W !!!!!!!\n", surplusPower)
		}
	}

	if totalConsumption > solarPowerTotal {
		// Discharge the battery:
		fmt.Printf("Power Deficit, discharging battery!\n")
		dischargePowerTotal = totalConsumption

		if forceCharging == false {
			for i := range inverters {
				inverters[i].ChargePower = 0
			}
		}
		//fmt.Printf("dischargePowerTotal: %v\n", dischargePowerTotal)
		//fmt.Printf("dischargePowerPerInverter: %v\n", dischargePowerPerInverter)
	}

	if batSoC < minSoC {
		fmt.Printf("Min SoC reached, do not discharge!\n")
		dischargePowerTotal = 0
	}

	if forceCharging == true {
		dischargePowerTotal = 0
	}

	if dischargePowerTotal > calcMaxDischargePowerBySoc() {
		fmt.Printf("limiting DischargePower due to SoC\n")
		dischargePowerTotal = calcMaxDischargePowerBySoc()
	}

	// No need to discharge the battery
	if dischargePowerTotal < 0 {
		dischargePowerTotal = 0
	}
	fmt.Printf("dischargePowerTotal: %v\n", dischargePowerTotal)

	// Calculate total deficit
	dischargePowerPerInverter := dischargePowerTotal / float32(len(inverters))
	for i := range inverters {
		// Limit max discharge power per inverter
		if dischargePowerPerInverter > 1500 {
			dischargePowerPerInverter = 1500
		}
		inverters[i].DischargePower = dischargePowerPerInverter
	}

	//j, _ := json.Marshal(inverters)
	//fmt.Printf("OVERVIEW: %v\n", string(j))

	for _, inverter := range inverters {
		j, _ := json.Marshal(inverter)
		fmt.Printf("OVERVIEW: %v\n", string(j))
	}
}
