package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	"github.com/brutella/hc/characteristic"
	"log"
)

type Switch struct {
	*service.Service

	On *characteristic.On
	Name *characteristic.Name
}

func NewSwitch(name string) *Switch {
	svc := Switch{}
	svc.Service = service.New(service.TypeSwitch)

	svc.On = characteristic.NewOn()
	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddCharacteristic(svc.On.Characteristic)
	svc.AddCharacteristic(svc.Name.Characteristic)

	return &svc
}

type Thermostat struct {
	*service.Service

	CurrentHeatingCoolingState *characteristic.CurrentHeatingCoolingState
	TargetHeatingCoolingState  *characteristic.TargetHeatingCoolingState
	CurrentTemperature         *characteristic.CurrentTemperature
	TargetTemperature          *characteristic.TargetTemperature
	TemperatureDisplayUnits    *characteristic.TemperatureDisplayUnits
	CurrentRelativeHumidity    *characteristic.CurrentRelativeHumidity
	TargetRelativeHumidity     *characteristic.TargetRelativeHumidity
}

func NewThermostat() *Thermostat {
	svc := Thermostat{}

	svc.Service = service.New(service.TypeThermostat)
	svc.CurrentHeatingCoolingState = characteristic.NewCurrentHeatingCoolingState()
	svc.AddCharacteristic(svc.CurrentHeatingCoolingState.Characteristic)
	svc.TargetHeatingCoolingState = characteristic.NewTargetHeatingCoolingState()
	svc.AddCharacteristic(svc.TargetHeatingCoolingState.Characteristic)
	svc.CurrentTemperature = characteristic.NewCurrentTemperature()
	svc.AddCharacteristic(svc.CurrentTemperature.Characteristic)
	svc.TargetTemperature = characteristic.NewTargetTemperature()
	svc.AddCharacteristic(svc.TargetTemperature.Characteristic)
	svc.TemperatureDisplayUnits = characteristic.NewTemperatureDisplayUnits()
	svc.AddCharacteristic(svc.TemperatureDisplayUnits.Characteristic)
	svc.CurrentRelativeHumidity = characteristic.NewCurrentRelativeHumidity()
	svc.AddCharacteristic(svc.CurrentRelativeHumidity.Characteristic)
	svc.TargetRelativeHumidity = characteristic.NewTargetRelativeHumidity()
	svc.AddCharacteristic(svc.TargetRelativeHumidity.Characteristic)
	return &svc
}

type FanV2 struct {
	*service.Service

	Active *characteristic.Active
	RotationSpeed *characteristic.RotationSpeed
}


func NewFanV2() *FanV2 {
	svc := FanV2{}
	svc.Service = service.New(service.TypeFanV2)

	svc.Active = characteristic.NewActive()
	svc.AddCharacteristic(svc.Active.Characteristic)
	svc.RotationSpeed = characteristic.NewRotationSpeed()
	svc.AddCharacteristic(svc.RotationSpeed.Characteristic)

	return &svc
}

type ClimateControl struct {
	*accessory.Accessory

	Thermostat	*Thermostat
	Fan		*FanV2
	Slat		*service.Slat
	Defrost		*Switch
	RearDefrost	*Switch
}

// NewClimateControl returns a ClimateControl which implements model.ClimateControl.
func NewClimateControl(info accessory.Info) *ClimateControl {


	acc := ClimateControl{}
	acc.Accessory = accessory.New(info, accessory.TypeAirConditioner)


	// adding the switch service
	acc.Defrost = NewSwitch("Defrost")
	acc.RearDefrost = NewSwitch("Rear Defrost")

	acc.AddService(acc.Defrost.Service)
	acc.AddService(acc.RearDefrost.Service)

	// adding the Thermostat service
	acc.Thermostat = NewThermostat()
	acc.Thermostat.CurrentTemperature.SetValue(25.0)
	acc.Thermostat.CurrentTemperature.SetMinValue(17.0)
	acc.Thermostat.CurrentTemperature.SetMaxValue(32.0)
	acc.Thermostat.CurrentTemperature.SetStepValue(0.1)
	acc.Thermostat.CurrentRelativeHumidity.SetStepValue(20)

	acc.Thermostat.TargetTemperature.SetValue(25.0)
	acc.Thermostat.TargetTemperature.SetMinValue(17.0)
	acc.Thermostat.TargetTemperature.SetMaxValue(32.0)
	acc.Thermostat.TargetTemperature.SetStepValue(0.1)
	acc.Thermostat.TargetRelativeHumidity.SetStepValue(20)


	acc.AddService(acc.Thermostat.Service)

	// adding the Fan service
	acc.Fan = NewFanV2()
	acc.AddService(acc.Fan.Service)

	// adding the Slat service
	acc.Slat = service.NewSlat()
	acc.AddService(acc.Slat.Service)

	return &acc
}

/*
func turnLightOn() {
	log.Println("Turn Light On")
}

func turnLightOff() {
	log.Println("Turn Light Off")
}
*/

func setTargetTemprature(acc *ClimateControl,targetTemp float64) {
	acc.Thermostat.CurrentTemperature.SetValue(targetTemp)
	log.Println("Set target temprature to ", targetTemp)
}

func main() {
	info := accessory.Info{
		Name:         "Climate Control",
		Model:         "2013",
		Manufacturer: "Ford",
	}

	acc := NewClimateControl(info)

/*
	acc.Lightbulb.On.OnValueRemoteUpdate(func(on bool) {
		if on == true {
			turnLightOn()
		} else {
			turnLightOff()
		}
	})
*/

	acc.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(targetTemp float64) {
			setTargetTemprature(acc, targetTemp)
	})

/*
	svc.CurrentHeatingCoolingState = characteristic.NewCurrentHeatingCoolingState()
	svc.AddCharacteristic(svc.CurrentHeatingCoolingState.Characteristic)

	svc.TargetHeatingCoolingState = characteristic.NewTargetHeatingCoolingState()
	svc.AddCharacteristic(svc.TargetHeatingCoolingState.Characteristic)

	svc.CurrentTemperature = characteristic.NewCurrentTemperature()
	svc.AddCharacteristic(svc.CurrentTemperature.Characteristic)

	svc.TargetTemperature = characteristic.NewTargetTemperature()
	svc.AddCharacteristic(svc.TargetTemperature.Characteristic)

	svc.TemperatureDisplayUnits = characteristic.NewTemperatureDisplayUnits()
	svc.AddCharacteristic(svc.TemperatureDisplayUnits.Characteristic)
*/


	t, err := hc.NewIPTransport(hc.Config{Pin: "32191123"}, acc.Accessory)
	if err != nil {
		log.Fatal(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
