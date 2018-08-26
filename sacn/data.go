package sacn

import (
	"fmt"
	"math"
)

const (
	vectorRootE131Data   = 4 //VECTOR_ROOT_E131_DATA
	vectorE131DataPacket = 2 //VECTOR_E131_DATA_PACKET
	vectorDmpSetProperty = 0x2
)

var constHeader = []byte{0, 0x10, 0, 0, 0x41, 0x53,
	0x43, 0x2d, 0x45, 0x31, 0x2e, 0x31, 0x37, 0x00, 0x00, 0x00}

//DataPacket is a byte array with unspecific length
type DataPacket struct {
	data   []byte
	length uint16
}

//NewDataPacket creates a new DataPacket with an empty 638-length byte slice
func NewDataPacket() DataPacket {
	p := DataPacket{make([]byte, 638), 126}
	//Set constants: at index [0;16[
	p.replace(0, constHeader)
	//Set vectors:
	p.replace(18, getAsBytes32(vectorRootE131Data))
	p.replace(40, getAsBytes32(vectorE131DataPacket))
	p.data[117] = vectorDmpSetProperty
	//set initial FAL
	p.setFAL(126)
	//set address and data type
	p.data[118] = 0xa1
	//set address increment
	p.data[122] = 0x1
	//Default priority:
	p.SetPriority(100)

	return p
}

//NewDataPacketRaw creates a new DataPacket based on the given raw bytes
func NewDataPacketRaw(raw []byte) (DataPacket, error) {
	var p DataPacket
	//Check the length of the raw bytes
	if len(raw) < 126 {
		return p, fmt.Errorf("The given raw bytes are too short! Min length is 126 was %v", len(raw))
	}
	p = NewDataPacket()
	//Make the array 638 long
	if len(raw) < 638 { //Add 0 if too short
		raw = append(raw, make([]byte, 638-len(raw))...)
	} else if len(raw) > 638 { //cut off the last bits if too long
		raw = raw[:638]
	}
	p.data = append([]byte(nil), raw...) //make a copy of the slice, we do not want to use a reference
	p.length = uint16(getAsUint32(raw[123:125]) + 125)
	return p, nil
}

//Set the FAL values in the byte slice according to the length
//Note: Length is the length of the whole message!
//Also sets the property value count!
//Also sets the length of the struct
func (d *DataPacket) setFAL(length uint16) {
	rootFAL := calculateFal(length - 16)
	d.replace(16, rootFAL[:])
	framingFAL := calculateFal(length - 38)
	d.replace(38, framingFAL[:])
	dmpFAL := calculateFal(length - 115)
	d.replace(115, dmpFAL[:])
	//property value count:
	propValCount := getAsBytes16(length - 125)
	d.replace(123, propValCount[:])

	d.length = length
}

//replace everything starting from the startindex in the datapacket with the given replacement
func (d *DataPacket) replace(startIndex int, replacement []byte) {
	d.data = append(d.data[:startIndex],
		append(replacement, d.data[len(replacement)+startIndex:]...)...)
}

//copy returns a copy of the DataPacket
func (d *DataPacket) copy() DataPacket {
	copySlice := make([]byte, len(d.data))
	copy(copySlice, d.data)
	return DataPacket{
		data:   copySlice,
		length: d.length,
	}
}

//SetCID sets the CID unique identifier
func (d *DataPacket) SetCID(cid [16]byte) {
	d.replace(22, cid[0:16])
}

//CID returns the cid that is set for this object
func (d *DataPacket) CID() [16]byte {
	tmpArray := [16]byte{}
	copy(tmpArray[:], d.data[22:38])
	return tmpArray
}

//SetSourceName sets the source name field to the given string values.
//Note that only the first 64 characters are used!
func (d *DataPacket) SetSourceName(s string) {
	b := [64]byte{}
	copy(b[:], []byte(s))
	d.replace(44, b[:64])
}

//SourceName returns the stored source name. Note that the source name max length is 64!
func (d *DataPacket) SourceName() string {
	i := 44 //the ending index for the string, because it is 0 terminated
	for i < 108 && d.data[i] != 0 {
		i++
	}
	return string(d.data[44:i])
}

//SetPriority sets the priority field for the packet. Value must be [0-200]!
func (d *DataPacket) SetPriority(prio byte) error {
	if prio > 200 {
		return fmt.Errorf("the priority was %v and therefore is not in range [0-200]", prio)
	}
	d.data[108] = prio
	return nil
}

//Priority returns the byte value of the priorty field of the packet. Value range: [0-200]
func (d *DataPacket) Priority() byte {
	return d.data[108]
}

//SetSyncAddress sets the synchronization universe for the given packet
func (d *DataPacket) SetSyncAddress(sync uint16) {
	d.replace(109, getAsBytes16(sync)[:])
}

//SyncAddress returns the sync universe of the given packet
func (d *DataPacket) SyncAddress() uint16 {
	return uint16(getAsUint32(d.data[109:111]))
}

//SetSequence sets the sequence number of the packet
func (d *DataPacket) SetSequence(sequ byte) {
	d.data[111] = sequ
}

//Sequence returns the sequence number of the packet
func (d *DataPacket) Sequence() byte {
	return d.data[111]
}

//SequenceIncr increments the sequence number
func (d *DataPacket) SequenceIncr() {
	d.data[111]++
}

//SetPreviewData sets the preview_data flag in this packet to the given value
func (d *DataPacket) SetPreviewData(value bool) {
	d.setOptionsBit(7, value)
}

//PreviewData returns wether this packet has the preview flag set
func (d *DataPacket) PreviewData() bool {
	return d.getOptionsBit(7)
}

//SetStreamTerminated sets the stream_termiantion falg on or off
func (d *DataPacket) SetStreamTerminated(value bool) {
	d.setOptionsBit(6, value)
}

//StreamTerminated returns the state of the stream_termination flag
func (d *DataPacket) StreamTerminated() bool {
	return d.getOptionsBit(6)
}

//SetForceSync sets the force_synchronization bit flag
func (d *DataPacket) SetForceSync(value bool) {
	d.setOptionsBit(5, value)
}

//ForceSync returns the state of the force_synchronization flag
func (d *DataPacket) ForceSync() bool {
	return d.getOptionsBit(5)
}

func (d *DataPacket) setOptionsBit(bit byte, value bool) {
	if value {
		d.data[112] = d.data[112] | byte(math.Pow(2, float64(bit)))
	} else {
		d.data[112] = d.data[112] & (byte(math.Pow(2, float64(bit))) ^ 0xFF)
	}
}

func (d *DataPacket) getOptionsBit(bit byte) bool {
	return d.data[112]&byte(math.Pow(2, float64(bit))) != 0
}

//SetUniverse sets the universe value of the packet
func (d *DataPacket) SetUniverse(universe uint16) {
	d.replace(113, getAsBytes16(universe))
}

//Universe returns the universe value of the packet
func (d *DataPacket) Universe() uint16 {
	return uint16(getAsUint32(d.data[113:115]))
}

//SetDmxStartCode sets the DMX start code that is transmitted together with the DMX data
func (d *DataPacket) SetDmxStartCode(startCode byte) {
	d.data[125] = startCode
}

//DmxStartCode return the start code of the given packet
func (d *DataPacket) DmxStartCode() byte {
	return d.data[125]
}

//SetData sets the dmx data for the given DataPacket
func (d *DataPacket) SetData(data []byte) {
	if len(data) > 512 {
		data = data[0:512]
	}
	//make the length a multiply of 2
	if len(data)%2 != 0 { //add a 0 to make the length sufficient
		data = append(data, 0)
	}
	d.setFAL(uint16(126 + len(data)))
	d.replace(126, data)
}

//Data returns the DMX data that is set for this DataPacket. Length: [0-512]
func (d *DataPacket) Data() []byte {
	return d.data[126:d.length]
}

func (d *DataPacket) getBytes() []byte {
	return d.data[:d.length]
}
