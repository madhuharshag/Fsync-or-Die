package kv

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
)

// Magic byte constants defining format boundaries.
var (
	FileHeaderMagic   = [8]byte{'K', 'I', 'P', 'D', 'B', '0', '0', '1'} // 8 bytes: KIPDB001
	TxnHeaderMagic    = [4]byte{'T', 'X', 'N', '1'}                     // 4 bytes: TXN1
	CommitFooterMagic = [8]byte{'C', 'O', 'M', 'M', 'I', 'T', '0', '1'} // 8 bytes: COMMIT01
)

const (
	// Fixed header and footer byte lengths.
	FileHeaderSize   = 32
	TxnHeaderSize    = 32
	CommitFooterSize = 32
	OpHeaderSize     = 7 // 1 byte kind + 2 bytes keyLen + 4 bytes valLen

	// Format versions.
	CurrentFormatVersion = 1
	CurrentTxnVersion    = 1
)

// FileHeader represents the immutable 32-byte header located at byte offset 0.
type FileHeader struct {
	Magic        [8]byte
	Version      uint32
	HeaderSize   uint32
	MaxFileBytes uint64
	Flags        uint32
	CRC          uint32
}

// TxnHeader represents the 32-byte header preceding every transaction payload.
type TxnHeader struct {
	Magic      [4]byte
	Version    uint16
	Flags      uint16
	TxID       uint64
	OpCount    uint32
	Reserved   uint32
	PayloadLen uint64
}

// CommitFooter represents the 32-byte footer finalizing every committed transaction.
type CommitFooter struct {
	Magic        [8]byte
	TxID         uint64
	TotalTxBytes uint64
	TxnCRC       uint32 // CRC32-IEEE over (TxnHeader 32 bytes + Payload)
	FooterCRC    uint32 // CRC32-IEEE over Footer bytes [0..27]
}

// NewFileHeader constructs a valid FileHeader with computed CRC.
func NewFileHeader(maxFileBytes uint64) FileHeader {
	h := FileHeader{
		Magic:        FileHeaderMagic,
		Version:      CurrentFormatVersion,
		HeaderSize:   FileHeaderSize,
		MaxFileBytes: maxFileBytes,
		Flags:        0,
	}
	h.CRC = CalculateFileHeaderCRC(h)
	return h
}

// EncodeFileHeader serializes a FileHeader into exactly 32 bytes (little-endian).
func EncodeFileHeader(h FileHeader) []byte {
	buf := make([]byte, FileHeaderSize)
	copy(buf[0:8], h.Magic[:])
	binary.LittleEndian.PutUint32(buf[8:12], h.Version)
	binary.LittleEndian.PutUint32(buf[12:16], h.HeaderSize)
	binary.LittleEndian.PutUint64(buf[16:24], h.MaxFileBytes)
	binary.LittleEndian.PutUint32(buf[24:28], h.Flags)
	binary.LittleEndian.PutUint32(buf[28:32], h.CRC)
	return buf
}

// CalculateFileHeaderCRC computes the CRC32-IEEE over bytes [0..27] of a FileHeader.
func CalculateFileHeaderCRC(h FileHeader) uint32 {
	buf := make([]byte, 28)
	copy(buf[0:8], h.Magic[:])
	binary.LittleEndian.PutUint32(buf[8:12], h.Version)
	binary.LittleEndian.PutUint32(buf[12:16], h.HeaderSize)
	binary.LittleEndian.PutUint64(buf[16:24], h.MaxFileBytes)
	binary.LittleEndian.PutUint32(buf[24:28], h.Flags)
	return crc32.ChecksumIEEE(buf)
}

// DecodeFileHeader parses and validates a 32-byte buffer into a FileHeader.
func DecodeFileHeader(buf []byte) (FileHeader, error) {
	if len(buf) < FileHeaderSize {
		return FileHeader{}, fmt.Errorf("%w: file header truncated (need 32 bytes, got %d)", ErrCorrupt, len(buf))
	}
	var h FileHeader
	copy(h.Magic[:], buf[0:8])
	if h.Magic != FileHeaderMagic {
		return FileHeader{}, fmt.Errorf("%w: invalid file header magic %q", ErrCorrupt, h.Magic)
	}
	h.Version = binary.LittleEndian.Uint32(buf[8:12])
	if h.Version != CurrentFormatVersion {
		return FileHeader{}, fmt.Errorf("%w: unsupported format version %d", ErrVersion, h.Version)
	}
	h.HeaderSize = binary.LittleEndian.Uint32(buf[12:16])
	if h.HeaderSize != FileHeaderSize {
		return FileHeader{}, fmt.Errorf("%w: invalid header size %d", ErrCorrupt, h.HeaderSize)
	}
	h.MaxFileBytes = binary.LittleEndian.Uint64(buf[16:24])
	h.Flags = binary.LittleEndian.Uint32(buf[24:28])
	h.CRC = binary.LittleEndian.Uint32(buf[28:32])

	expectedCRC := crc32.ChecksumIEEE(buf[0:28])
	if h.CRC != expectedCRC {
		return FileHeader{}, fmt.Errorf("%w: file header CRC mismatch (stored %08x, computed %08x)", ErrCorrupt, h.CRC, expectedCRC)
	}
	return h, nil
}

// EncodeTxnHeader serializes a TxnHeader into exactly 32 bytes (little-endian).
func EncodeTxnHeader(h TxnHeader) []byte {
	buf := make([]byte, TxnHeaderSize)
	copy(buf[0:4], h.Magic[:])
	binary.LittleEndian.PutUint16(buf[4:6], h.Version)
	binary.LittleEndian.PutUint16(buf[6:8], h.Flags)
	binary.LittleEndian.PutUint64(buf[8:16], h.TxID)
	binary.LittleEndian.PutUint32(buf[16:20], h.OpCount)
	binary.LittleEndian.PutUint32(buf[20:24], h.Reserved)
	binary.LittleEndian.PutUint64(buf[24:32], h.PayloadLen)
	return buf
}

// DecodeTxnHeader parses and validates a 32-byte buffer into a TxnHeader.
func DecodeTxnHeader(buf []byte) (TxnHeader, error) {
	if len(buf) < TxnHeaderSize {
		return TxnHeader{}, fmt.Errorf("%w: transaction header truncated (need 32 bytes, got %d)", ErrCorrupt, len(buf))
	}
	var h TxnHeader
	copy(h.Magic[:], buf[0:4])
	if h.Magic != TxnHeaderMagic {
		return TxnHeader{}, fmt.Errorf("%w: invalid transaction header magic %q", ErrCorrupt, h.Magic)
	}
	h.Version = binary.LittleEndian.Uint16(buf[4:6])
	if h.Version != CurrentTxnVersion {
		return TxnHeader{}, fmt.Errorf("%w: unsupported transaction version %d", ErrVersion, h.Version)
	}
	h.Flags = binary.LittleEndian.Uint16(buf[6:8])
	if h.Flags != 0 {
		return TxnHeader{}, fmt.Errorf("%w: nonzero transaction flags %d", ErrVersion, h.Flags)
	}
	h.TxID = binary.LittleEndian.Uint64(buf[8:16])
	h.OpCount = binary.LittleEndian.Uint32(buf[16:20])
	if h.OpCount == 0 || h.OpCount > MaxBatchOps {
		return TxnHeader{}, fmt.Errorf("%w: operation count out of range (1..%d): %d", ErrInvalid, MaxBatchOps, h.OpCount)
	}
	h.Reserved = binary.LittleEndian.Uint32(buf[20:24])
	if h.Reserved != 0 {
		return TxnHeader{}, fmt.Errorf("%w: nonzero transaction reserved field %d", ErrCorrupt, h.Reserved)
	}
	h.PayloadLen = binary.LittleEndian.Uint64(buf[24:32])
	return h, nil
}

// EncodeOpHeader encodes the 7-byte metadata preceding each key-value payload.
func EncodeOpHeader(kind Kind, keyLen uint16, valLen uint32) []byte {
	buf := make([]byte, OpHeaderSize)
	buf[0] = byte(kind)
	binary.LittleEndian.PutUint16(buf[1:3], keyLen)
	binary.LittleEndian.PutUint32(buf[3:7], valLen)
	return buf
}

// DecodeOpHeader extracts (Kind, keyLen, valLen) from a 7-byte buffer.
func DecodeOpHeader(buf []byte) (Kind, uint16, uint32, error) {
	if len(buf) < OpHeaderSize {
		return 0, 0, 0, fmt.Errorf("%w: operation header truncated (need 7 bytes, got %d)", ErrCorrupt, len(buf))
	}
	k := Kind(buf[0])
	if k != PutOp && k != DeleteOp {
		return 0, 0, 0, fmt.Errorf("%w: invalid operation kind %d", ErrCorrupt, k)
	}
	keyLen := binary.LittleEndian.Uint16(buf[1:3])
	valLen := binary.LittleEndian.Uint32(buf[3:7])
	if keyLen < MinKeyLen || int(keyLen) > MaxKeyLen {
		return 0, 0, 0, fmt.Errorf("%w: key length %d out of bounds (1..%d)", ErrInvalid, keyLen, MaxKeyLen)
	}
	if k == DeleteOp && valLen != 0 {
		return 0, 0, 0, fmt.Errorf("%w: delete operation must have zero value length, got %d", ErrInvalid, valLen)
	}
	if k == PutOp && valLen > MaxValueLen {
		return 0, 0, 0, fmt.Errorf("%w: put value length %d exceeds maximum %d", ErrInvalid, valLen, MaxValueLen)
	}
	return k, keyLen, valLen, nil
}

// ValidateOp verifies constraints on a single mutation operation.
func ValidateOp(op Op) error {
	if op.Kind != PutOp && op.Kind != DeleteOp {
		return fmt.Errorf("%w: invalid operation kind %d", ErrInvalid, op.Kind)
	}
	if len(op.Key) < MinKeyLen || len(op.Key) > MaxKeyLen {
		return fmt.Errorf("%w: key length %d out of range [%d, %d]", ErrInvalid, len(op.Key), MinKeyLen, MaxKeyLen)
	}
	if op.Kind == DeleteOp {
		if len(op.Value) != 0 {
			return fmt.Errorf("%w: delete operation must have zero-length value", ErrInvalid)
		}
	} else { // PutOp
		if len(op.Value) > MaxValueLen {
			return fmt.Errorf("%w: value length %d exceeds maximum %d", ErrInvalid, len(op.Value), MaxValueLen)
		}
	}
	return nil
}

// ValidateOps checks an entire batch of operations.
func ValidateOps(ops []Op) error {
	if len(ops) == 0 {
		return fmt.Errorf("%w: batch is empty", ErrInvalid)
	}
	if len(ops) > MaxBatchOps {
		return fmt.Errorf("%w: batch operation count %d exceeds maximum %d", ErrInvalid, len(ops), MaxBatchOps)
	}
	for i, op := range ops {
		if err := ValidateOp(op); err != nil {
			return fmt.Errorf("operation %d invalid: %w", i, err)
		}
	}
	return nil
}

// EncodeOp serializes a single Op into [1B kind][2B klen][4B vlen][key][val].
func EncodeOp(op Op) ([]byte, error) {
	if err := ValidateOp(op); err != nil {
		return nil, err
	}
	keyLen := len(op.Key)
	valLen := len(op.Value)
	totalLen := OpHeaderSize + keyLen + valLen
	buf := make([]byte, totalLen)
	buf[0] = byte(op.Kind)
	binary.LittleEndian.PutUint16(buf[1:3], uint16(keyLen))
	binary.LittleEndian.PutUint32(buf[3:7], uint32(valLen))
	copy(buf[7:7+keyLen], op.Key)
	if valLen > 0 {
		copy(buf[7+keyLen:], op.Value)
	}
	return buf, nil
}

// DecodeOp decodes an Op from a buffer, returning the Op, total bytes consumed, and any error.
func DecodeOp(buf []byte) (Op, int, error) {
	kind, keyLen, valLen, err := DecodeOpHeader(buf)
	if err != nil {
		return Op{}, 0, err
	}
	reqLen := OpHeaderSize + int(keyLen) + int(valLen)
	if len(buf) < reqLen {
		return Op{}, 0, fmt.Errorf("%w: truncated operation payload (need %d bytes, got %d)", ErrCorrupt, reqLen, len(buf))
	}
	keyStart := OpHeaderSize
	keyEnd := keyStart + int(keyLen)
	valStart := keyEnd
	valEnd := valStart + int(valLen)

	keyCopy := make([]byte, keyLen)
	copy(keyCopy, buf[keyStart:keyEnd])

	var valCopy []byte
	if valLen > 0 {
		valCopy = make([]byte, valLen)
		copy(valCopy, buf[valStart:valEnd])
	}
	return Op{
		Kind:  kind,
		Key:   keyCopy,
		Value: valCopy,
	}, reqLen, nil
}

// EncodeCommitFooter encodes a CommitFooter into exactly 32 bytes (little-endian).
func EncodeCommitFooter(f CommitFooter) []byte {
	buf := make([]byte, CommitFooterSize)
	copy(buf[0:8], f.Magic[:])
	binary.LittleEndian.PutUint64(buf[8:16], f.TxID)
	binary.LittleEndian.PutUint64(buf[16:24], f.TotalTxBytes)
	binary.LittleEndian.PutUint32(buf[24:28], f.TxnCRC)
	binary.LittleEndian.PutUint32(buf[28:32], f.FooterCRC)
	return buf
}

// CalculateFooterCRC computes the CRC32-IEEE over bytes [0..27] of a CommitFooter.
func CalculateFooterCRC(magic [8]byte, txID uint64, totalTxBytes uint64, txnCRC uint32) uint32 {
	buf := make([]byte, 28)
	copy(buf[0:8], magic[:])
	binary.LittleEndian.PutUint64(buf[8:16], txID)
	binary.LittleEndian.PutUint64(buf[16:24], totalTxBytes)
	binary.LittleEndian.PutUint32(buf[24:28], txnCRC)
	return crc32.ChecksumIEEE(buf)
}

// NewCommitFooter builds a CommitFooter and automatically populates FooterCRC.
func NewCommitFooter(txID uint64, totalTxBytes uint64, txnCRC uint32) CommitFooter {
	return CommitFooter{
		Magic:        CommitFooterMagic,
		TxID:         txID,
		TotalTxBytes: totalTxBytes,
		TxnCRC:       txnCRC,
		FooterCRC:    CalculateFooterCRC(CommitFooterMagic, txID, totalTxBytes, txnCRC),
	}
}

// DecodeCommitFooter parses and verifies a 32-byte commit footer buffer.
func DecodeCommitFooter(buf []byte) (CommitFooter, error) {
	if len(buf) < CommitFooterSize {
		return CommitFooter{}, fmt.Errorf("%w: commit footer truncated (need 32 bytes, got %d)", ErrCorrupt, len(buf))
	}
	var f CommitFooter
	copy(f.Magic[:], buf[0:8])
	if f.Magic != CommitFooterMagic {
		return CommitFooter{}, fmt.Errorf("%w: invalid commit footer magic %q", ErrCorrupt, f.Magic)
	}
	f.TxID = binary.LittleEndian.Uint64(buf[8:16])
	f.TotalTxBytes = binary.LittleEndian.Uint64(buf[16:24])
	f.TxnCRC = binary.LittleEndian.Uint32(buf[24:28])
	f.FooterCRC = binary.LittleEndian.Uint32(buf[28:32])

	expectedFooterCRC := crc32.ChecksumIEEE(buf[0:28])
	if f.FooterCRC != expectedFooterCRC {
		return CommitFooter{}, fmt.Errorf("%w: commit footer CRC mismatch (stored %08x, computed %08x)", ErrCorrupt, f.FooterCRC, expectedFooterCRC)
	}
	return f, nil
}

// CalculatePayloadSize calculates the exact encoded payload bytes for a list of operations,
// preventing integer overflow.
func CalculatePayloadSize(ops []Op) (uint64, error) {
	var total uint64
	for i, op := range ops {
		kLen := uint64(len(op.Key))
		vLen := uint64(len(op.Value))
		opBytes := uint64(OpHeaderSize) + kLen + vLen
		if math.MaxUint64-total < opBytes {
			return 0, fmt.Errorf("%w: payload length overflow at operation %d", ErrInvalid, i)
		}
		total += opBytes
	}
	return total, nil
}

// CalculateTotalTxnSize calculates the complete transaction length:
// TxnHeaderSize (32) + PayloadBytes + CommitFooterSize (32).
func CalculateTotalTxnSize(ops []Op) (uint64, error) {
	payloadLen, err := CalculatePayloadSize(ops)
	if err != nil {
		return 0, err
	}
	overhead := uint64(TxnHeaderSize + CommitFooterSize) // 64 bytes
	if math.MaxUint64-payloadLen < overhead {
		return 0, fmt.Errorf("%w: total transaction size exceeds uint64 max", ErrInvalid)
	}
	return payloadLen + overhead, nil
}
