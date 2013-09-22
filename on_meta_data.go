package flvprobe

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
)

type on_meta_data struct {
	// common properties
	duration          float64
	width             int64
	height            int64
	video_data_rate   float64
	frame_rate        float64
	video_codec_id    int64
	audio_data_rate   float64
	audio_delay       float64
	audio_sample_rate float64
	audio_sample_size int64
	can_seek_to_end   byte
	creation          string
	stereo            byte
	audio_codec_id    int64
	file_size         int64

	last_timestamp          float64
	last_keyframe_location  int64
	last_keyframe_timestamp float64
	creator                 string
	metadatacreator         string
	has_keyframes           byte
	has_video               byte
	has_audio               byte
	has_metadata            byte

	data_size  int64
	video_size int64
	audio_size int64
	positions  []int64
	timestamps []float64
}

const (
	value_type_number = iota
	value_type_byte
	value_type_string
	value_type_object
	value_type_movieclip
	value_type_null
	value_type_undefined
	value_type_reference
	value_type_ecma
	value_type_eof
	value_type_strictarray
	value_type_date
	value_type_longstring
)

func skip_script_data_string(reader io.ReadSeeker) (err error) {
	var w uint16
	err = binary.Read(reader, binary.BigEndian, &w)
	if err == nil {
		_, err = reader.Seek(int64(w), 1)
	}
	return
}
func skip_script_data_longstring(reader io.ReadSeeker) (err error) {
	var l uint32
	err = binary.Read(reader, binary.BigEndian, &l)
	if err == nil {
		_, err = reader.Seek(int64(l), 1)
	}
	return
}
func skip_script_data_ecma(reader io.ReadSeeker) (err error) {
	var cnt int32
	err = binary.Read(reader, binary.BigEndian, &cnt)
	skip_script_data_objectproperties(reader)
	return
}

func script_data_ints(reader io.Reader) ([]int64, error) {
	var b byte
	binary.Read(reader, binary.BigEndian, &b) // 10

	var cnt int32
	err := binary.Read(reader, binary.BigEndian, &cnt)
	v := make([]int64, cnt)
	for i := 0; i < int(cnt) && err == nil; i++ {
		v[i], err = script_data_int(reader)
	}
	return v, err
}
func script_data_numbers(reader io.Reader) ([]float64, error) {
	// strict array
	var b byte
	binary.Read(reader, binary.BigEndian, &b) // 10

	var cnt int32
	err := binary.Read(reader, binary.BigEndian, &cnt)
	v := make([]float64, cnt)
	for i := 0; i < int(cnt) && err == nil; i++ {
		v[i], err = script_data_number(reader)
	}
	return v, err
}

func skip_script_data_strictarray(reader io.ReadSeeker) (err error) {
	var cnt int32
	err = binary.Read(reader, binary.BigEndian, &cnt)
	for i := 0; i < int(cnt) && err == nil; i++ {
		err = skip_script_data_value(reader)
	}
	return
}
func skip_script_data_objectproperties(reader io.ReadSeeker) (err error) {
	for err == nil {
		err = skip_script_data_objectproperty(reader)
	}
	return nil
}

func skip_script_data_objectproperty(reader io.ReadSeeker) (err error) {
	skip_script_data_string(reader)
	err = skip_script_data_value(reader)
	return
}

func decode_keyframes(reader io.ReadSeeker, v on_meta_data) (err error) {
	var t byte
	err = binary.Read(reader, binary.BigEndian, &t) // 3
	for err == nil {
		n, _ := script_data_string(reader)
		switch n {
		default:
			err = skip_script_data_value(reader)
		case `filepositions`:
			v.positions, err = script_data_ints(reader)
		case `times`:
			v.timestamps, err = script_data_numbers(reader)
		}
	}
	return nil
}

func skip_script_data_value(reader io.ReadSeeker) (err error) {
	var t byte
	err = binary.Read(reader, binary.BigEndian, &t)
	//	log.Println(t, `skip value type`)
	switch t {
	case value_type_number:
		_, err = reader.Seek(8, 1)
	case value_type_byte:
		_, err = reader.Seek(1, 1)
	case value_type_string:
		err = skip_script_data_string(reader)
	case value_type_object:
		err = skip_script_data_objectproperties(reader)
	case value_type_movieclip:
		err = skip_script_data_string(reader)
	case value_type_null:
	case value_type_undefined:
	case value_type_reference:
	case value_type_ecma:
		err = skip_script_data_ecma(reader)
	case value_type_eof:
		err = io.EOF
		// do nothing
	case value_type_strictarray:
		err = skip_script_data_strictarray(reader)
	case value_type_date:
		_, err = reader.Seek(8, 1) //DateTime
		_, err = reader.Seek(2, 1) // localdatetimeoffset
	case value_type_longstring:
		err = skip_script_data_longstring(reader)
	}
	return
}

func decode_on_meta_data(reader io.ReadSeeker) (v on_meta_data, err error) {
	skip_script_data_value(reader) // name
	_, err = reader.Seek(1, 1)     // 8 , ecma
	_, err = reader.Seek(4, 1)     //size
	for err == nil {
		var n string
		n, err = script_data_string(reader)
		switch n {
		default:
			log.Println(n, `skipped prop`)
			err = skip_script_data_value(reader)
		case `creator`: //string
			v.creator, err = script_data_type_string(reader)
		case `metadatacreator`: // string
			v.metadatacreator, err = script_data_type_string(reader)
		case `hasKeyframes`: // byte
			v.has_keyframes, err = script_data_bool(reader)
		case `hasVideo`: // byte
			v.has_video, err = script_data_bool(reader)
		case `hasAudio`: // byte
			v.has_audio, err = script_data_bool(reader)
		case `hasMetadata`: // byte
			v.has_metadata, err = script_data_bool(reader)
		case `datasize`: //number
			v.data_size, err = script_data_int(reader)
		case `videosize`: //num
			v.video_size, err = script_data_int(reader)
		case `audiosize`: //num
			v.audio_size, err = script_data_int(reader)
		case `lasttimestamp`: // num
			v.last_timestamp, err = script_data_number(reader)
		case `lastkeyframetimestamp`: // num
			v.last_keyframe_timestamp, err = script_data_number(reader)
		case `lastkeyframelocation`: //num
			v.last_keyframe_location, err = script_data_int(reader)
		case `keyframes`: //object
			err = decode_keyframes(reader, v)
		case `audiocodecid`:
			v.audio_codec_id, err = script_data_int(reader)
		case `audiodatarate`:
			v.audio_data_rate, err = script_data_number(reader)
		case `audiodelay`:
			v.audio_delay, err = script_data_number(reader)
		case `audiosamplerate`:
			v.audio_sample_rate, err = script_data_number(reader)
		case `audiosamplesize`:
			v.audio_sample_size, err = script_data_int(reader)
		case `canSeekToEnd`:
			v.can_seek_to_end, err = script_data_bool(reader)
		case `creationdate`:
			var b byte
			binary.Read(reader, binary.BigEndian, &b) // byte
			v.creation, err = script_data_string(reader)
		case `duration`:
			v.duration, err = script_data_number(reader)
		case `filesize`:
			v.file_size, err = script_data_int(reader)
		case `framerate`:
			v.frame_rate, err = script_data_number(reader)
		case `height`:
			v.height, err = script_data_int(reader)
		case `stereo`:
			v.stereo, err = script_data_bool(reader)
		case `videocodecid`:
			v.video_codec_id, err = script_data_int(reader)
		case `videodatarate`:
			v.video_data_rate, err = script_data_number(reader)
		case `width`:
			v.width, err = script_data_int(reader)
		case ``:
			err = script_data_objectendmarker(reader)
		}
	}
	return
}

func script_data_type_string(reader io.Reader) (string, error) {
	var b byte
	binary.Read(reader, binary.BigEndian, &b)
	return script_data_string(reader)
}
func script_data_string(reader io.Reader) (string, error) {
	var l uint16
	err := binary.Read(reader, binary.BigEndian, &l)
	b := make([]byte, l)
	if err == nil {
		err = binary.Read(reader, binary.BigEndian, &b)
	}
	v := string(b)
	return v, err
}

func script_data_bool(reader io.Reader) (byte, error) {
	var b byte
	err := binary.Read(reader, binary.BigEndian, &b) // byte
	err = binary.Read(reader, binary.BigEndian, &b)
	return b, err
}

func script_data_int(reader io.Reader) (int64, error) {
	v, err := script_data_number(reader)
	return int64(v), err
}
func script_data_number(reader io.Reader) (float64, error) {
	var b byte
	err := binary.Read(reader, binary.BigEndian, &b) // byte
	var d float64
	err = binary.Read(reader, binary.BigEndian, &d)
	return d, err
}
func script_data_objectendmarker(reader io.Reader) error {
	var b byte
	binary.Read(reader, binary.BigEndian, &b)
	if b == 9 {
		return io.EOF
	}
	return fmt.Errorf(`encounter invalid object-end-marker '%v'`, b)
}
