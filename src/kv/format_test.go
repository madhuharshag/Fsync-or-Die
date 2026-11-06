package kv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"
)

func TestFixedLayoutSizes(t *testing.T) {
	if FileHeaderSize != 32 {
		t.Fatalf("expected FileHeaderSize=32, got %d", FileHeaderSize)
	}
	if TxnHeaderSize != 32 {
		t.Fatalf("expected TxnHeaderSize=32, got %d", TxnHeaderSize)
	}
	if CommitFooterSize != 32 {
		t.Fatalf("expected CommitFooterSize=32, got %d", CommitFooterSize)
	}
	if OpHeaderSize != 7 {
		t.Fatalf("expected OpHeaderSize=7, got %d", OpHeaderSize)
	}
}

func TestFileHeader_EncodingAndOffsets(t *testing.T) {
	maxBytes := uint64(10 * 1024 * 1024)
	h := NewFileHeader(maxBytes)

	encoded := EncodeFileHeader(h)
	if len(encoded) != 32 {
		t.Fatalf("expected 32 bytes encoded, got %d", len(encoded))
	}

	// Verify exact field offsets
	if !bytes.Equal(encoded[0:8], []byte("KIPDB001")) {
		t.Fatalf("unexpected magic: %v", encoded[0:8])
	}
	if v := binary.LittleEndian.Uint32(encoded[8:12]); v != 1 {
		t.Fatalf("expected version 1, got %d", v)
	}
	if sz := binary.LittleEndian.Uint32(encoded[12:16]); sz != 32 {
		t.Fatalf("expected header size 32, got %d", sz)
	}
	if mb := binary.LittleEndian.Uint64(encoded[16:24]); mb != maxBytes {
		t.Fatalf("expected maxBytes %d, got %d", maxBytes, mb)
	}
	if flags := binary.LittleEndian.Uint32(encoded[24:28]); flags != 0 {
		t.Fatalf("expected flags 0, got %d", flags)
	}
	storedCRC := binary.LittleEndian.Uint32(encoded[28:32])
	computedCRC := crc32.ChecksumIEEE(encoded[0:28])
	if storedCRC != computedCRC {
		t.Fatalf("CRC mismatch: stored %08x, computed %08x", storedCRC, computedCRC)
	}

	// Decode and verify roundtrip
	decoded, err := DecodeFileHeader(encoded)
	if err != nil {
		t.Fatalf("failed to decode valid file header: %v", err)
	}
	if decoded.Magic != FileHeaderMagic {
		t.Errorf("magic mismatch: %v vs %v", decoded.Magic, FileHeaderMagic)
	}
	if decoded.Version != 1 || decoded.HeaderSize != 32 || decoded.MaxFileBytes != maxBytes || decoded.Flags != 0 {
		t.Errorf("decoded field mismatch: %+v", decoded)
	}
	if decoded.CRC != storedCRC {
		t.Errorf("CRC mismatch in decoded struct: %08x vs %08x", decoded.CRC, storedCRC)
	}
}

func TestFileHeader_CorruptAndTruncated(t *testing.T) {
	h := NewFileHeader(1024)
	encoded := EncodeFileHeader(h)

	// Truncated
	for sz := 0; sz < 32; sz++ {
		if _, err := DecodeFileHeader(encoded[:sz]); !errors.Is(err, ErrCorrupt) {
			t.Fatalf("expected ErrCorrupt for size %d, got %v", sz, err)
		}
	}

	// Corrupt magic
	badMagic := make([]byte, 32)
	copy(badMagic, encoded)
	badMagic[0] = 'X'
	if _, err := DecodeFileHeader(badMagic); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("expected ErrCorrupt on bad magic, got %v", err)
	}

	// Corrupt CRC
	badCRC := make([]byte, 32)
	copy(badCRC, encoded)
	badCRC[31] ^= 0xFF
	if _, err := DecodeFileHeader(badCRC); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("expected ErrCorrupt on bit flip in CRC, got %v", err)
	}
}

func TestTxnHeader_EncodingAndValidation(t *testing.T) {
	orig := TxnHeader{
		Magic:      TxnHeaderMagic,
		Version:    1,
		Flags:      0,
		TxID:       42,
		OpCount:    5,
		Reserved:   0,
		PayloadLen: 1842,
	}

	encoded := EncodeTxnHeader(orig)
	if len(encoded) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(encoded))
	}

	// Check exact offsets
	if !bytes.Equal(encoded[0:4], []byte("TXN1")) {
		t.Fatalf("unexpected txn magic %v", encoded[0:4])
	}
	if v := binary.LittleEndian.Uint16(encoded[4:6]); v != 1 {
		t.Fatalf("unexpected txn version %d", v)
	}
	if id := binary.LittleEndian.Uint64(encoded[8:16]); id != 42 {
		t.Fatalf("unexpected txID %d", id)
	}
	if count := binary.LittleEndian.Uint32(encoded[16:20]); count != 5 {
		t.Fatalf("unexpected opCount %d", count)
	}
	if plen := binary.LittleEndian.Uint64(encoded[24:32]); plen != 1842 {
		t.Fatalf("unexpected payloadLen %d", plen)
	}

	// Decode roundtrip
	decoded, err := DecodeTxnHeader(encoded)
	if err != nil {
		t.Fatalf("failed to decode valid txn header: %v", err)
	}
	if decoded != orig {
		t.Fatalf("decoded txn header %+v does not match orig %+v", decoded, orig)
	}

	// Reject OpCount = 0
	zeroOps := orig
	zeroOps.OpCount = 0
	if _, err := DecodeTxnHeader(EncodeTxnHeader(zeroOps)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid for OpCount=0, got %v", err)
	}

	// Reject OpCount > MaxBatchOps
	overflowOps := orig
	overflowOps.OpCount = MaxBatchOps + 1
	if _, err := DecodeTxnHeader(EncodeTxnHeader(overflowOps)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid for OpCount>MaxBatchOps, got %v", err)
	}
}

func TestOp_EncodingDecodingAndLimits(t *testing.T) {
	tests := []struct {
		name    string
		op      Op
		wantErr error
	}{
		{
			name: "valid put",
			op: Op{
				Kind:  PutOp,
				Key:   []byte("account/balance"),
				Value: []byte("100500"),
			},
			wantErr: nil,
		},
		{
			name: "valid put with empty value",
			op: Op{
				Kind:  PutOp,
				Key:   []byte("flag/exists"),
				Value: []byte{},
			},
			wantErr: nil,
		},
		{
			name: "valid delete",
			op: Op{
				Kind:  DeleteOp,
				Key:   []byte("temporary/key"),
				Value: nil,
			},
			wantErr: nil,
		},
		{
			name: "valid binary key and value",
			op: Op{
				Kind:  PutOp,
				Key:   []byte{0x00, 0xFF, 0xFE, 0x01},
				Value: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			},
			wantErr: nil,
		},
		{
			name: "invalid zero-length key",
			op: Op{
				Kind:  PutOp,
				Key:   []byte{},
				Value: []byte("val"),
			},
			wantErr: ErrInvalid,
		},
		{
			name: "invalid oversized key",
			op: Op{
				Kind:  PutOp,
				Key:   make([]byte, MaxKeyLen+1),
				Value: []byte("val"),
			},
			wantErr: ErrInvalid,
		},
		{
			name: "delete with non-empty value",
			op: Op{
				Kind:  DeleteOp,
				Key:   []byte("k"),
				Value: []byte("fail"),
			},
			wantErr: ErrInvalid,
		},
		{
			name: "invalid kind",
			op: Op{
				Kind:  Kind(99),
				Key:   []byte("k"),
				Value: nil,
			},
			wantErr: ErrInvalid,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := EncodeOp(tc.op)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected encode error: %v", err)
			}

			// Verify OpHeader layout: [1B kind][2B klen][4B vlen]
			kind := Kind(enc[0])
			klen := binary.LittleEndian.Uint16(enc[1:3])
			vlen := binary.LittleEndian.Uint32(enc[3:7])
			if kind != tc.op.Kind || int(klen) != len(tc.op.Key) || int(vlen) != len(tc.op.Value) {
				t.Fatalf("encoded header values incorrect: kind=%d klen=%d vlen=%d", kind, klen, vlen)
			}

			// Decode and verify roundtrip
			dec, consumed, err := DecodeOp(enc)
			if err != nil {
				t.Fatalf("unexpected decode error: %v", err)
			}
			if consumed != len(enc) {
				t.Fatalf("consumed %d bytes, expected %d", consumed, len(enc))
			}
			if dec.Kind != tc.op.Kind {
				t.Errorf("kind mismatch: %d vs %d", dec.Kind, tc.op.Kind)
			}
			if !bytes.Equal(dec.Key, tc.op.Key) {
				t.Errorf("key mismatch: %v vs %v", dec.Key, tc.op.Key)
			}
			if len(tc.op.Value) > 0 && !bytes.Equal(dec.Value, tc.op.Value) {
				t.Errorf("value mismatch: %v vs %v", dec.Value, tc.op.Value)
			}
		})
	}
}

func TestCommitFooter_EncodingAndCRC(t *testing.T) {
	txID := uint64(9999)
	totalTxBytes := uint64(1024)
	txnCRC := uint32(0x12345678)

	footer := NewCommitFooter(txID, totalTxBytes, txnCRC)
	encoded := EncodeCommitFooter(footer)

	if len(encoded) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(encoded))
	}

	// Verify exact offsets
	if !bytes.Equal(encoded[0:8], []byte("COMMIT01")) {
		t.Fatalf("invalid commit footer magic %v", encoded[0:8])
	}
	if id := binary.LittleEndian.Uint64(encoded[8:16]); id != txID {
		t.Fatalf("expected txID %d, got %d", txID, id)
	}
	if b := binary.LittleEndian.Uint64(encoded[16:24]); b != totalTxBytes {
		t.Fatalf("expected totalTxBytes %d, got %d", totalTxBytes, b)
	}
	if crc := binary.LittleEndian.Uint32(encoded[24:28]); crc != txnCRC {
		t.Fatalf("expected txnCRC %08x, got %08x", txnCRC, crc)
	}
	storedFooterCRC := binary.LittleEndian.Uint32(encoded[28:32])
	computedFooterCRC := crc32.ChecksumIEEE(encoded[0:28])
	if storedFooterCRC != computedFooterCRC {
		t.Fatalf("footer CRC mismatch: stored %08x, computed %08x", storedFooterCRC, computedFooterCRC)
	}

	// Decode roundtrip
	decoded, err := DecodeCommitFooter(encoded)
	if err != nil {
		t.Fatalf("failed to decode valid commit footer: %v", err)
	}
	if decoded != footer {
		t.Fatalf("decoded footer %+v does not match %+v", decoded, footer)
	}

	// Mismatched footer CRC
	corruptBuf := make([]byte, 32)
	copy(corruptBuf, encoded)
	corruptBuf[28] ^= 0x01
	if _, err := DecodeCommitFooter(corruptBuf); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("expected ErrCorrupt on flipped footer CRC byte, got %v", err)
	}
}

func TestCalculateTransactionSizes(t *testing.T) {
	ops := []Op{
		{Kind: PutOp, Key: []byte("rice"), Value: []byte("500")}, // 7 + 4 + 3 = 14 bytes
	}

	payloadLen, err := CalculatePayloadSize(ops)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payloadLen != 14 {
		t.Fatalf("expected payloadLen 14, got %d", payloadLen)
	}

	totalLen, err := CalculateTotalTxnSize(ops)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 32 (TxnHeader) + 14 (payload) + 32 (CommitFooter) = 78 bytes (matches example from brief)
	if totalLen != 78 {
		t.Fatalf("expected totalLen 78 bytes, got %d", totalLen)
	}
}
