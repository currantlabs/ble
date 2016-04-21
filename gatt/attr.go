package gatt

import (
	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/uuid"
)

func genAttr(ss []*Service, base uint16) *att.Range {
	var svrRange []att.Attribute
	h := base
	for i, s := range ss {
		var svcRanger []att.Attribute
		h, svcRanger = genSvcAttr(s, h)
		if i == len(ss)-1 {
			svcRanger[0].EndingHandle = 0xFFFF
		}
		svrRange = append(svrRange, svcRanger...)
	}
	att.DumpAttributes(svrRange)
	return &att.Range{Attributes: svrRange, Base: base}
}

func genSvcAttr(s *Service, h uint16) (uint16, []att.Attribute) {
	s.h = h
	a := att.Attribute{
		Handle: h,
		Type:   uuid.UUID(attrPrimaryServiceUUID),
		Value:  s.UUID(),
	}
	h++
	svcRange := []att.Attribute{a}

	for _, c := range s.Characteristics() {
		var charRange []att.Attribute
		h, charRange = genCharAttr(c, h)
		svcRange = append(svcRange, charRange...)
	}

	svcRange[0].EndingHandle = h - 1
	return h, svcRange
}

func genCharAttr(c *Characteristic, h uint16) (uint16, []att.Attribute) {
	c.h = h
	c.vh = h + 1
	valueh := c.vh
	ca := att.Attribute{
		Handle: h,
		Type:   uuid.UUID(attrCharacteristicUUID),
		Value:  append([]byte{byte(c.Properties()), byte(valueh), byte((valueh) >> 8)}, c.UUID()...),
	}
	va := att.Attribute{
		Handle:       valueh,
		EndingHandle: valueh,
		Type:         c.UUID(),
		Value:        c.value.v,
		Pvt:          c.value,
	}
	h += 2

	charRange := []att.Attribute{ca, va}
	for _, d := range c.Descriptors() {
		charRange = append(charRange, genDescAttr(d, h))
		h++
	}

	charRange[0].EndingHandle = h - 1
	return h, charRange
}

func genDescAttr(d *Descriptor, h uint16) att.Attribute {
	d.h = h
	return att.Attribute{
		Handle:       h,
		EndingHandle: h,
		Type:         d.UUID(),
		Value:        d.value.v,
		Pvt:          d.value,
	}
}
