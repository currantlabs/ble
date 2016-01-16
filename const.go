package bt

// HCI Packet types
const (
	pktTypeCommand uint8 = 0x01
	pktTypeACLData uint8 = 0x02
	pktTypeSCOData uint8 = 0x03
	pktTypeEvent   uint8 = 0x04
	pktTypeVendor  uint8 = 0xFF
)

// Event Types [Vol 6 Part B, 2.3 Advertising PDU, 4.4.2].
const (
	advInd        = 0x00 // Connectable undirected advertising (ADV_IND).
	advDirectInd  = 0x01 // Connectable directed advertising (ADV_DIRECT_IND).
	advScanInd    = 0x02 // Scannable undirected advertising (ADV_SCAN_IND).
	advNonconnInd = 0x03 // Non connectable undirected advertising (ADV_NONCONN_IND).
	scanRsp       = 0x04 // Scan Response (SCAN_RSP).
)

// Packet boundary flags of HCI ACL Data Packet [Vol 2, Part E, 5.4.2].
const (
	pbfHostToControllerStart = 0x00 // Start of a non-automatically-flushable from host to controller.
	pbfContinuing            = 0x01 // Continuing fragment.
	pbfControllerToHostStart = 0x02 // Start of a non-automatically-flushable from controller to host.
	pbfCompleteL2CAPPDU      = 0x03 // A automatically flushable complete PDU. (Not used in LE-U).
)

// L2CAP Channel Identifier namespace for LE-U logical link [Vol 3, Part A, 2.1].
const (
	cidLEAtt    uint16 = 0x04 // Attribute Protocol [Vol 3, Part F].
	cidLESignal uint16 = 0x05 // Low Energy L2CAP Signaling channel [Vol 3, Part A, 4].
	cidLESMP    uint16 = 0x06 // SecurityManager Protocol [Vol 3, Part H].
)

const (
	roleMaster = 0x00
	roleSlave  = 0x01
)
