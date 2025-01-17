package lnwire

import (
	"bytes"
	"io"

	"github.com/barebitcoin/zpay32/tlv"
	"github.com/btcsuite/btcd/btcec/v2"
)

// OnionPacketSize is the size of the serialized Sphinx onion packet included
// in each UpdateAddHTLC message. The breakdown of the onion packet is as
// follows: 1-byte version, 33-byte ephemeral public key (for ECDH), 1300-bytes
// of per-hop data, and a 32-byte HMAC over the entire packet.
const OnionPacketSize = 1366

type (
	// BlindingPointTlvType is the type for ephemeral pubkeys used in
	// route blinding.
	BlindingPointTlvType = tlv.TlvType0

	// BlindingPointRecord holds an optional blinding point on update add
	// htlc.
	//nolint:lll
	BlindingPointRecord = tlv.OptionalRecordT[BlindingPointTlvType, *btcec.PublicKey]
)

// UpdateAddHTLC is the message sent by Alice to Bob when she wishes to add an
// HTLC to his remote commitment transaction. In addition to information
// detailing the value, the ID, expiry, and the onion blob is also included
// which allows Bob to derive the next hop in the route. The HTLC added by this
// message is to be added to the remote node's "pending" HTLCs.  A subsequent
// CommitSig message will move the pending HTLC to the newly created commitment
// transaction, marking them as "staged".
type UpdateAddHTLC struct {
	// ChanID is the particular active channel that this UpdateAddHTLC is
	// bound to.
	ChanID ChannelID

	// ID is the identification server for this HTLC. This value is
	// explicitly included as it allows nodes to survive single-sided
	// restarts. The ID value for this sides starts at zero, and increases
	// with each offered HTLC.
	ID uint64

	// Amount is the amount of millisatoshis this HTLC is worth.
	Amount MilliSatoshi

	// PaymentHash is the payment hash to be included in the HTLC this
	// request creates. The pre-image to this HTLC must be revealed by the
	// upstream peer in order to fully settle the HTLC.
	PaymentHash [32]byte

	// Expiry is the number of blocks after which this HTLC should expire.
	// It is the receiver's duty to ensure that the outgoing HTLC has a
	// sufficient expiry value to allow her to redeem the incoming HTLC.
	Expiry uint32

	// OnionBlob is the raw serialized mix header used to route an HTLC in
	// a privacy-preserving manner. The mix header is defined currently to
	// be parsed as a 4-tuple: (groupElement, routingInfo, headerMAC,
	// body).  First the receiving node should use the groupElement, and
	// its current onion key to derive a shared secret with the source.
	// Once the shared secret has been derived, the headerMAC should be
	// checked FIRST. Note that the MAC only covers the routingInfo field.
	// If the MAC matches, and the shared secret is fresh, then the node
	// should strip off a layer of encryption, exposing the next hop to be
	// used in the subsequent UpdateAddHTLC message.
	OnionBlob [OnionPacketSize]byte

	// BlindingPoint is the ephemeral pubkey used to optionally blind the
	// next hop for this htlc.
	BlindingPoint BlindingPointRecord

	// ExtraData is the set of data that was appended to this message to
	// fill out the full maximum transport message size. These fields can
	// be used to specify optional data such as custom TLV fields.
	ExtraData ExtraOpaqueData
}

// NewUpdateAddHTLC returns a new empty UpdateAddHTLC message.
func NewUpdateAddHTLC() *UpdateAddHTLC {
	return &UpdateAddHTLC{}
}

// A compile time check to ensure UpdateAddHTLC implements the lnwire.Message
// interface.
var _ Message = (*UpdateAddHTLC)(nil)

// Decode deserializes a serialized UpdateAddHTLC message stored in the passed
// io.Reader observing the specified protocol version.
//
// This is part of the lnwire.Message interface.
func (c *UpdateAddHTLC) Decode(r io.Reader, pver uint32) error {
	if err := ReadElements(r,
		&c.ChanID,
		&c.ID,
		&c.Amount,
		c.PaymentHash[:],
		&c.Expiry,
		c.OnionBlob[:],
		&c.ExtraData,
	); err != nil {
		return err
	}

	blindingRecord := c.BlindingPoint.Zero()
	tlvMap, err := c.ExtraData.ExtractRecords(&blindingRecord)
	if err != nil {
		return err
	}

	if val, ok := tlvMap[c.BlindingPoint.TlvType()]; ok && val == nil {
		c.BlindingPoint = tlv.SomeRecordT(blindingRecord)
	}

	// Set extra data to nil if we didn't parse anything out of it so that
	// we can use assert.Equal in tests.
	if len(tlvMap) == 0 {
		c.ExtraData = nil
	}

	return nil
}

// Encode serializes the target UpdateAddHTLC into the passed io.Writer
// observing the protocol version specified.
//
// This is part of the lnwire.Message interface.
func (c *UpdateAddHTLC) Encode(w *bytes.Buffer, pver uint32) error {
	if err := WriteChannelID(w, c.ChanID); err != nil {
		return err
	}

	if err := WriteUint64(w, c.ID); err != nil {
		return err
	}

	if err := WriteMilliSatoshi(w, c.Amount); err != nil {
		return err
	}

	if err := WriteBytes(w, c.PaymentHash[:]); err != nil {
		return err
	}

	if err := WriteUint32(w, c.Expiry); err != nil {
		return err
	}

	if err := WriteBytes(w, c.OnionBlob[:]); err != nil {
		return err
	}

	// Only include blinding point in extra data if present.
	var records []tlv.RecordProducer

	c.BlindingPoint.WhenSome(func(b tlv.RecordT[BlindingPointTlvType,
		*btcec.PublicKey]) {

		records = append(records, &b)
	})

	err := EncodeMessageExtraData(&c.ExtraData, records...)
	if err != nil {
		return err
	}

	return WriteBytes(w, c.ExtraData)
}

// MsgType returns the integer uniquely identifying this message type on the
// wire.
//
// This is part of the lnwire.Message interface.
func (c *UpdateAddHTLC) MsgType() MessageType {
	return MsgUpdateAddHTLC
}

// TargetChanID returns the channel id of the link for which this message is
// intended.
//
// NOTE: Part of peer.LinkUpdater interface.
func (c *UpdateAddHTLC) TargetChanID() ChannelID {
	return c.ChanID
}
