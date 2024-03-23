package typesv1

import (
	"encoding/binary"
	"io/fs"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func FileInfoFromFS(fi fs.FileInfo) *FileInfo {
	return &FileInfo{
		Name:    fi.Name(),
		Size:    uint64(fi.Size()),
		Mode:    uint32(fi.Mode()),
		ModTime: timestamppb.New(fi.ModTime()),
		IsDir:   fi.IsDir(),
	}
}

func Uint256FromArray32Byte(arr [32]byte) *Uint256 {
	return &Uint256{
		A: binary.LittleEndian.Uint64(arr[0:8]),
		B: binary.LittleEndian.Uint64(arr[8:16]),
		C: binary.LittleEndian.Uint64(arr[16:24]),
		D: binary.LittleEndian.Uint64(arr[24:32]),
	}
}

func Array32ByteFromUint256(v *Uint256) [32]byte {
	var out [32]byte
	out[0+0] = byte(v.A)
	out[0+1] = byte(v.A >> 8)
	out[0+2] = byte(v.A >> 16)
	out[0+3] = byte(v.A >> 24)
	out[0+4] = byte(v.A >> 32)
	out[0+5] = byte(v.A >> 40)
	out[0+6] = byte(v.A >> 48)
	out[0+7] = byte(v.A >> 56)
	out[8+0] = byte(v.B)
	out[8+1] = byte(v.B >> 8)
	out[8+2] = byte(v.B >> 16)
	out[8+3] = byte(v.B >> 24)
	out[8+4] = byte(v.B >> 32)
	out[8+5] = byte(v.B >> 40)
	out[8+6] = byte(v.B >> 48)
	out[8+7] = byte(v.B >> 56)
	out[16+0] = byte(v.C)
	out[16+1] = byte(v.C >> 8)
	out[16+2] = byte(v.C >> 16)
	out[16+3] = byte(v.C >> 24)
	out[16+4] = byte(v.C >> 32)
	out[16+5] = byte(v.C >> 40)
	out[16+6] = byte(v.C >> 48)
	out[16+7] = byte(v.C >> 56)
	out[24+0] = byte(v.D)
	out[24+1] = byte(v.D >> 8)
	out[24+2] = byte(v.D >> 16)
	out[24+3] = byte(v.D >> 24)
	out[24+4] = byte(v.D >> 32)
	out[24+5] = byte(v.D >> 40)
	out[24+6] = byte(v.D >> 48)
	out[24+7] = byte(v.D >> 56)
	return out
}
