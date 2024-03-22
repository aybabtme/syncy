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
