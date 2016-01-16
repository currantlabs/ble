package gatt

import (
	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/uuid"
)

func generateAttributes(ss []*Service, base uint16) *att.Range {
	var svrRange []att.Attribute
	h := base
	for i, s := range ss {
		var svcRanger []att.Attribute
		h, svcRanger = generateServiceAttributes(s, h)
		if i == len(ss)-1 {
			svcRanger[0].EndingHandle = 0xFFFF
		}
		svrRange = append(svrRange, svcRanger...)
	}
	att.DumpAttributes(svrRange)
	return &att.Range{Attributes: svrRange, Base: base}
}

func generateServiceAttributes(s *Service, h uint16) (uint16, []att.Attribute) {
	s.h = h
	a := att.Attribute{
		Handle: h,
		Type:   uuid.UUID(attrPrimaryServiceUUID),
		Value:  s.UUID,
	}
	h++
	svcRange := []att.Attribute{a}

	for _, c := range s.Characteristics {
		var charRange []att.Attribute
		h, charRange = generateCharAttributes(c, h)
		svcRange = append(svcRange, charRange...)
	}

	svcRange[0].EndingHandle = h - 1
	return h, svcRange
}

func generateCharAttributes(c *Characteristic, h uint16) (uint16, []att.Attribute) {
	c.h = h
	c.vh = h + 1
	valueh := c.vh
	ca := att.Attribute{
		Handle: h,
		Type:   uuid.UUID(attrCharacteristicUUID),
		Value:  append([]byte{byte(c.Property), byte(valueh), byte((valueh) >> 8)}, c.UUID...),
	}
	va := att.Attribute{
		Handle:       valueh,
		EndingHandle: valueh,
		Type:         uuid.UUID(c.UUID),
		Pvt:          &c.value,
	}
	h += 2

	charRange := []att.Attribute{ca, va}
	for _, d := range c.Descriptors {
		charRange = append(charRange, generateDescAttributes(d, h))
		h++
	}

	charRange[0].EndingHandle = h - 1
	return h, charRange
}

func generateDescAttributes(d *Descriptor, h uint16) att.Attribute {
	d.h = h
	return att.Attribute{
		Handle:       h,
		EndingHandle: h,
		Type:         uuid.UUID(d.UUID),
		Pvt:          &d.value,
	}
}
