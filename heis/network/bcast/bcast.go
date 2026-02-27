package bcast

import (
	"Sanntid/Heis/Network/conn"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
)

// størrelse på buffer for sending og mottak av meldinger 
const bufSize = 1024

// Encodes received values from `chans` into type-tagged JSON, then broadcasts
// it on `port`

//mål: broadcaste melding til en port. Det som broadcastes er innhold som ligger på en channel.
//hvordan: Den konvertere til JSON og sende på nettverk for oss på port.  
//ut: TypeId, JSON
func Transmitter(port int, chans ...interface{}) { //portnummer som den sender til, en eller flere channels som skal lyttes på 
	typeNames := make([]string, len(chans))
	selectCases := make([]reflect.SelectCase, len(typeNames))
	//lytter på de ulike kanalene og lagrer type 
	for i, ch := range chans {
		selectCases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		}
		typeNames[i] = reflect.TypeOf(ch).Elem().String()
	}

	conn := conn.DialBroadcastUDP(port)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port)) //her settes porten som den skap på 

	//løkke for å vente på meldinger, gjøre om til json og sende på nettverk
	for {
		chosen, value, _ := reflect.Select(selectCases) //kanal som har fått en melding, og innholdet i meldingen
		jsonstr, _ := json.Marshal(value.Interface()) 
		ttj, _ := json.Marshal(typeTaggedJSON{
			TypeId: typeNames[chosen],
			JSON:   jsonstr,
		})
		if len(ttj) > bufSize {
		    panic(fmt.Sprintf(
		        "Tried to send a message longer than the buffer size (length: %d, buffer size: %d)\n\t'%s'\n"+
		        "Either send smaller packets, or go to network/bcast/bcast.go and increase the buffer size",
		        len(ttj), bufSize, string(ttj)))
		}
		conn.WriteTo(ttj, addr)
    		
	}
}


//mål: receiver lytter på nettet og putter meldinger inn i riktige go channels. F.eks for å lytte på state til andre heiser 
//inn: TypeId, JSON
//hvordan: lytte på port, lese og tolke meldinger, gjøre om json tilbake, sende til korrekt channel 

// Matches type-tagged JSON received on `port` to element types of `chans`, then
// sends the decoded value on the corresponding channel
func Receiver(port int, chans ...interface{}) {
	checkArgs(chans...) //sjekker at vi sender inn gyldige kanaler 
	chansMap := make(map[string]interface{})
	//finne kanalen den skal sende til ved å se på type, og lagre i map
	for _, ch := range chans {
		chansMap[reflect.TypeOf(ch).Elem().String()] = ch
	}
	//lager buffer og åpnet UDP 
	var buf [bufSize]byte
	conn := conn.DialBroadcastUDP(port)
	for {
		n, _, e := conn.ReadFrom(buf[0:])
		if e != nil {
			fmt.Printf("bcast.Receiver(%d, ...):ReadFrom() failed: \"%+v\"\n", port, e)
		}

		var ttj typeTaggedJSON
		json.Unmarshal(buf[0:n], &ttj)
		ch, ok := chansMap[ttj.TypeId]
		if !ok {
			continue
		}
		v := reflect.New(reflect.TypeOf(ch).Elem())
		json.Unmarshal(ttj.JSON, v.Interface())
		reflect.Select([]reflect.SelectCase{{
			Dir:  reflect.SelectSend,
			Chan: reflect.ValueOf(ch),
			Send: reflect.Indirect(v),
		}})
	}
}

type typeTaggedJSON struct {
	TypeId string
	JSON   []byte
}


//-----------

// Den sjekker at du bruker Transmitter og Receiver riktig før programmet starter. Krav er kanalaer av ulike type, og at de kan konvertere til JSON 

//----------
// Checks that args to Tx'er/Rx'er are valid:
//  All args must be channels
//  Element types of channels must be encodable with JSON
//  No element types are repeated
// Implementation note:
//  - Why there is no `isMarshalable()` function in encoding/json is a mystery,
//    so the tests on element type are hand-copied from `encoding/json/encode.go`
func checkArgs(chans ...interface{}) {
	n := 0
	for range chans {
		n++
	}
	elemTypes := make([]reflect.Type, n)

	for i, ch := range chans {
		// Must be a channel
		if reflect.ValueOf(ch).Kind() != reflect.Chan {
			panic(fmt.Sprintf(
				"Argument must be a channel, got '%s' instead (arg# %d)",
				reflect.TypeOf(ch).String(), i+1))
		}

		elemType := reflect.TypeOf(ch).Elem()

		// Element type must not be repeated
		for j, e := range elemTypes {
			if e == elemType {
				panic(fmt.Sprintf(
					"All channels must have mutually different element types, arg# %d and arg# %d both have element type '%s'",
					j+1, i+1, e.String()))
			}
		}
		elemTypes[i] = elemType

		// Element type must be encodable with JSON
		checkTypeRecursive(elemType, []int{i+1})

	}
}


//sjekker om en type kan konverteres til JSON. Trenger ikke å endres 

func checkTypeRecursive(val reflect.Type, offsets []int){
	switch val.Kind() {
	case reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		panic(fmt.Sprintf(
			"Channel element type must be supported by JSON, got '%s' instead (nested arg# %v)",
			val.String(), offsets))
	case reflect.Map:
		if val.Key().Kind() != reflect.String {
			panic(fmt.Sprintf(
				"Channel element type must be supported by JSON, got '%s' instead (map keys must be 'string') (nested arg# %v)",
				val.String(), offsets))
		}
		checkTypeRecursive(val.Elem(), offsets)
	case reflect.Array, reflect.Ptr, reflect.Slice:
		checkTypeRecursive(val.Elem(), offsets)
	case reflect.Struct:
		for idx := 0; idx < val.NumField(); idx++ {
			checkTypeRecursive(val.Field(idx).Type, append(offsets, idx+1))
		}
	}
}
