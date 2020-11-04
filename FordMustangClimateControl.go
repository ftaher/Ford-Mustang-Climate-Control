package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
	"github.com/brutella/hc/characteristic"
	"log"
	"unsafe"
	"math"
//	"fmt" 
	"github.com/brutella/can"
	"net"
	"os"
	"os/signal"
)


var firstByte, fifthByte uint8
var ACon, ACmax, ACrecirculated, ACdefrost, ACrearDefrost bool
var FanFront, FanDown, FanWindow bool
var TargetTemperature, FanSpeed float64

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
	Defrost		*Switch
	RearDefrost	*Switch
}


// NewClimateControl returns a ClimateControl which implements model.ClimateControl.
func NewClimateControl(info accessory.Info) *ClimateControl {


	acc1 := ClimateControl{}
	acc1.Accessory = accessory.New(info, accessory.TypeAirConditioner)


	// adding the switch service
	acc1.Defrost = NewSwitch("Defrost")
	acc1.RearDefrost = NewSwitch("Rear Defrost")

	acc1.AddService(acc1.Defrost.Service)
	acc1.AddService(acc1.RearDefrost.Service)

	// adding the Thermostat service
	acc1.Thermostat = NewThermostat()
	acc1.Thermostat.CurrentTemperature.SetValue(25.0)
	acc1.Thermostat.CurrentTemperature.SetMinValue(15.0)
	acc1.Thermostat.CurrentTemperature.SetMaxValue(32.0)
	acc1.Thermostat.CurrentTemperature.SetStepValue(0.5)

	acc1.Thermostat.TargetTemperature.SetValue(25.0)
	acc1.Thermostat.TargetTemperature.SetMinValue(15.0)
	acc1.Thermostat.TargetTemperature.SetMaxValue(32.0)
	acc1.Thermostat.TargetTemperature.SetStepValue(0.5)


	acc1.AddService(acc1.Thermostat.Service)

	// adding the Fan service
	acc1.Fan = NewFanV2()
	acc1.Fan.RotationSpeed.SetValue(10.0)
	acc1.Fan.RotationSpeed.SetMinValue(10.0)
	acc1.Fan.RotationSpeed.SetMaxValue(70.0)
	acc1.Fan.RotationSpeed.SetStepValue(10.0)
	acc1.AddService(acc1.Fan.Service)

	return &acc1
}

func setTargetTemprature(acc *ClimateControl,targetTemp float64) {
	if targetTemp < 16 {
		FanSpeed = 30.0
		acc.Thermostat.CurrentHeatingCoolingState.SetValue(2)
		acc.Thermostat.TargetHeatingCoolingState.SetValue(2)
		log.Println("MAX Cooling with the highest Speed")
	} else if targetTemp < 17 {
		FanSpeed = 20.0
		acc.Thermostat.CurrentHeatingCoolingState.SetValue(2)
		acc.Thermostat.TargetHeatingCoolingState.SetValue(2)
		log.Println("MAX Cooling with the 2nd highest Speed")
	} else if targetTemp == 17 {
		FanSpeed = 10.0
		acc.Thermostat.CurrentHeatingCoolingState.SetValue(2)
		acc.Thermostat.TargetHeatingCoolingState.SetValue(2)
		log.Println("MAX Cooling with the lowest fan Speed")
	} else {
		FanSpeed = 10.0
		log.Println("Set target temprature to ", targetTemp)
	}
	TargetTemperature = targetTemp
}
func setFanSpeed(acc *ClimateControl,targetSpeed float64) {
	FanSpeed = targetSpeed
	acc.Fan.RotationSpeed.SetValue(targetSpeed)
	log.Println("Set Fan Speed to ", targetSpeed)
}
func setCoolingState(acc *ClimateControl,coolingState int) {
	if coolingState == 0 { //Turn off Climate Control 398#4000000000000000
		if firstByte != 0 || fifthByte != 0 { SendCANFrame(uint8(0x40)) }
		log.Println("off => Turning OFF the ClimateControl")
	} else if coolingState == 1 { // Turn off the ACon 398#0400000000000000
		if !ACon { SendCANFrame(uint8(0x04)) }
		log.Println("Heating => Turning OFF the AC")
	} else if coolingState == 2 { // Turn on the ACmax 398#0200000000000000
		if !ACmax { SendCANFrame(uint8(0x02)) }
		log.Println("Cooling => Turning ON ACmax")
	} else if coolingState == 3 { // Turn off the ACrecirculated 398#0100000000000000
		if ACrecirculated { SendCANFrame(uint8(0x01)) }
		log.Println("Auto => Turning off the ACrecirculated")
	} else { // Print an Error message
		log.Println("ERROR, Set Cooling State to ", coolingState)
	}
}
func setRearDefrost(acc *ClimateControl) { // 398#0800000000000000
	SendCANFrame(uint8(0x08))
	log.Println("Toggle Rear Defrost ")
}
func setDefrost(acc *ClimateControl) { // 398#1000000000000000
	SendCANFrame(uint8(0x10))
	log.Println("Toggle Rear Defrost ")
}

var FCC *ClimateControl
var bus *can.Bus

func main() {
	info := accessory.Info{
		Name:         "Climate Control",
		Model:         "2013",
		Manufacturer: "Ford",
	}

	FCC1 := NewClimateControl(info)
	FCC =  FCC1


	FCC1.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(targetTemp float64) {
			setTargetTemprature(FCC1, targetTemp)
	})
	FCC1.Thermostat.TargetHeatingCoolingState.OnValueRemoteUpdate(func(coolingState int) {
			setCoolingState(FCC1, coolingState)
	})
	FCC1.Fan.RotationSpeed.OnValueRemoteUpdate(func(targetSpeed float64) {
			setFanSpeed(FCC1, targetSpeed)
	})
	FCC1.RearDefrost.On.OnValueRemoteUpdate(func(bool) {
			setRearDefrost(FCC1)
	})
	FCC1.Defrost.On.OnValueRemoteUpdate(func(bool) {
			setDefrost(FCC1)
	})

	/* Setting up the can bus */
	iface, err := net.InterfaceByName("can0")

	if err != nil {
		log.Fatalf("Could not find network interface can0 (%v)", err)
	}

	conn, err := can.NewReadWriteCloserForInterface(iface)

	if err != nil {
		log.Fatal(err)
	}

	bus = can.NewBus(conn)
	bus.SubscribeFunc(NewCANFrame)


	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	go func() {
		select {
		case <-c:
			bus.Disconnect()
			os.Exit(1)
		}
	}()

	go bus.ConnectAndPublish()


	/* Creating the Homekit Transport */
	t, err := hc.NewIPTransport(hc.Config{Pin: "32191124"}, FCC1.Accessory)
	if err != nil {
		log.Fatal(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()

}

// NewCANFrame capture any new CAN frame
func NewCANFrame(frm can.Frame) {

	log.Printf("Got a CAN frame: %-4x",frm.Data[0])
	if frm.ID == uint32(0x387) {
		firstByte = frm.Data[0]
		fifthByte = frm.Data[4]
	}

	/*
	# 387#00000000F0000000 #lights for the 5th byte

      00010000 => 10  => AC on
      00100000 => 20  => Down
      01000000 => 40  => front
      10000000 => 80  => Window
	*/
	if fifthByte & uint8(0x10) > 0 { ACon = true } else { ACon = false }
	if fifthByte & uint8(0x20) > 0 { FanDown = true } else { FanDown = false }
	if fifthByte & uint8(0x40) > 0 { FanFront = true } else { FanFront = false }
	if fifthByte & uint8(0x80) > 0 { FanWindow = true } else {FanWindow = false }
	/*
	# 387#FF00000000000000 #lights for the 1st byte

      00000100 => 04  => recycle on
      00001000 => 08  => AC Max on
      00010000 => 10  => Rear Defrost on
      00100000 => 20  => Front Defrost on
	*/
	// ACrecirculated
	if firstByte & uint8(0x04) > 0 { ACrecirculated = true } else { ACrecirculated = false }
	if firstByte & uint8(0x08) > 0 { ACmax = true } else { ACmax = false }
	if firstByte & uint8(0x10) > 0 { ACrearDefrost = true } else { ACrearDefrost = false }
	if firstByte & uint8(0x20) > 0 { ACdefrost = true } else { ACdefrost = false }

	/* Cases:
		Off:  all lights are off
		Heat: ACon is off
		Cool: ACMax is on
		Auto: ACon is On But both ACMAX and ACrecirculated are off
	*/

	if firstByte == 0 && fifthByte == 0 {
		FCC.Thermostat.CurrentHeatingCoolingState.SetValue(0)
		FCC.Thermostat.TargetHeatingCoolingState.SetValue(0)
	} else if ACon == false {
		FCC.Thermostat.CurrentHeatingCoolingState.SetValue(1)
		FCC.Thermostat.TargetHeatingCoolingState.SetValue(1)
	} else if ACmax == true {
		FCC.Thermostat.CurrentHeatingCoolingState.SetValue(2)
		FCC.Thermostat.TargetHeatingCoolingState.SetValue(2)
	} else if ACrecirculated == false {
		FCC.Thermostat.CurrentHeatingCoolingState.SetValue(3)
		FCC.Thermostat.TargetHeatingCoolingState.SetValue(3)
	}
	FCC.Thermostat.CurrentTemperature.SetValue(TargetTemperature)
	FCC.Thermostat.TargetTemperature.SetValue(TargetTemperature)

	FCC.RearDefrost.On.SetValue(ACrearDefrost)
	FCC.Defrost.On.SetValue(ACdefrost)

	SendCANFrame(uint8(0x00))
}

func SendCANFrame(code uint8) {
	var TargetTemp float64
	if TargetTemperature < 17 { TargetTemp = 17.0 } else {TargetTemp = TargetTemperature}
	uint16TargetTemp := uint16((TargetTemp - 17.0) * 18)
	uint8TargetTemp := *(*[2]uint8)(unsafe.Pointer(&uint16TargetTemp))
	uintFanSpeed := uint8((math.Floor(FanSpeed/10)-1)*20)

	// Temperature => Byte number 2 and 3
	// FanSpeed => Byte number 5
	zero := uint8(0x00)
	data := [8]uint8{code, uint8TargetTemp[1], uint8TargetTemp[0], zero, uintFanSpeed, zero,zero,zero }

	frm := can.Frame{
	ID:     0x398,
	Length: 8,
	Flags:  0,
	Res0:   0,
	Res1:   0,
	//Data:   [8]uint8{0x05}}
	Data:   data }

	bus.Publish(frm)
}
