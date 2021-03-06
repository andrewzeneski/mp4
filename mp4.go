package mp4

import (
	"io"
	"math"
	"strconv"
	"time"
)

// A MPEG-4 media
//
// A MPEG-4 media contains three main boxes :
//
//   ftyp : the file type box
//   moov : the movie box (meta-data)
//   mdat : the media data (chunks and samples)
//
// Other boxes can also be present (pdin, moof, mfra, free, ...), but are not decoded.
type MP4 struct {
	Ftyp  *FtypBox
	Moov  *MoovBox
	Mdat  *MdatBox
	boxes []Box
}

// Decode decodes a media from a Reader
func Decode(r io.Reader) (*MP4, error) {
	v := &MP4{
		boxes: []Box{},
	}
LoopBoxes:
	for {
		h, err := DecodeHeader(r)
		if err != nil {
			return nil, err
		}
		box, err := DecodeBox(h, r)
		if err != nil {
			return nil, err
		}
		switch h.Type {
		case "ftyp":
			v.Ftyp = box.(*FtypBox)
		case "moov":
			v.Moov = box.(*MoovBox)
		case "mdat":
			v.Mdat = box.(*MdatBox)
			v.Mdat.ContentSize = h.Size - BoxHeaderSize
			break LoopBoxes
		default:
			v.boxes = append(v.boxes, box)
		}
	}
	return v, nil
}

// Dump displays some information about a media
func (m *MP4) Dump() {
	m.Ftyp.Dump()
	m.Moov.Dump()
}

// Boxes lists the top-level boxes from a media
func (m *MP4) Boxes() []Box {
	return m.boxes
}

// Encode encodes a media to a Writer
func (m *MP4) Encode(w io.Writer) error {
	err := m.Ftyp.Encode(w)
	if err != nil {
		return err
	}
	err = m.Moov.Encode(w)
	if err != nil {
		return err
	}
	for _, b := range m.boxes {
		err = b.Encode(w)
		if err != nil {
			return err
		}
	}
	err = m.Mdat.Encode(w)
	if err != nil {
		return err
	}
	return nil
}

func (m *MP4) Size() (size int) {
	size += m.Ftyp.Size()
	size += m.Moov.Size()
	size += m.Mdat.Size()

	for _, b := range m.Boxes() {
		size += b.Size()
	}

	return
}

func (m *MP4) Duration() time.Duration {
	return time.Second * time.Duration(m.Moov.Mvhd.Duration) / time.Duration(m.Moov.Mvhd.Timescale)
}

func (m *MP4) VideoDimensions() (int, int) {
	for _, trak := range m.Moov.Trak {
		h, _ := strconv.ParseFloat(trak.Tkhd.Height.String(), 64)
		w, _ := strconv.ParseFloat(trak.Tkhd.Width.String(), 64)
		if h > 0 && w > 0 {
			return int(math.Floor(w)), int(math.Floor(h))
		}
	}
	return 0, 0
}

func (m *MP4) AudioVolume() float64 {
	for _, trak := range m.Moov.Trak {
		vol, _ := strconv.ParseFloat(trak.Tkhd.Volume.String(), 64)
		if vol > 0 {
			return vol
		}
	}
	return 0.0
}
