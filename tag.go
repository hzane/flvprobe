package flvprobe

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

func TraverseFlv(reader io.ReadSeeker) (err error) {
	h := expect_flv_header(reader)
	if !h.validate() {
		fmt.Println(h.sig(), `invalid flv format`)
		return
	}
	var omd on_meta_data
	fmt.Println(h, `flv-header`)
	for err == nil {
		_, err = next_previous_tag_size(reader)
		var th flv_tag_header
		th, err = next_tag_header(reader)
		switch th.tag_type {
		case tag_type_audio:
			_, err = reader.Seek(int64(th.data_size.to_uint32()), 1)
		case tag_type_video:
			_, err = reader.Seek(int64(th.data_size.to_uint32()), 1)
		case tag_type_script:
			//			fmt.Println(`enter script tag`)
			omd, err = decode_on_meta_data(reader)
		default:
			err = fmt.Errorf(`unknown tag type %v`, th.tag_type)
		}
	}

	fmt.Println(`duration`, omd.duration,
		"\nwidth", omd.width,
		"\nheight", omd.height,
		"\nvideo-rate", omd.video_data_rate,
		"\nframe-rate", omd.frame_rate,
		"\nvideo-codec", omd.video_codec_id,
		"\naudio-rate", omd.audio_data_rate,
		"\naudio-delay", omd.audio_delay,
		"\naudio-sample-rate", omd.audio_sample_rate,
		"\naudio-sample-size", omd.audio_sample_size,
		"\ncan-seek", omd.can_seek_to_end,
		"\ncreation", omd.creation,
		"\nstereo", omd.stereo,
		"\naudio-codec", omd.audio_codec_id,
		"\nfile-size", omd.file_size,

		"\nlast-times", omd.last_timestamp,
		"\nlast-keyf-loc", omd.last_keyframe_location,
		"\nlast-keyf-times", omd.last_keyframe_timestamp,
		"\ncreator", omd.creator,
		"\nmeta-creator", omd.metadatacreator,
		"\nhas-keyf", omd.has_keyframes,
		"\nhas-video", omd.has_video,
		"\nhas-audio", omd.has_audio,
		"\nhas-meta", omd.has_metadata,

		"\ndata-size", omd.data_size,
		"\nvideo-size", omd.video_size,
		"\naudio-size", omd.audio_size)
	return err
}

type flv_header struct { // size is 9 bytes
	signature   [3]byte // 'FLV'
	version     byte
	flags       byte // 00000x0x  audio-video
	data_offset uint32
}

func (this flv_header) validate() bool {
	s := strings.ToUpper(this.sig())
	return s == `FLV` && this.data_offset == 9
}

func (this flv_header) sig() string {
	return string(this.signature[:])
}
func expect_flv_header(reader io.Reader) flv_header {
	var v flv_header
	binary.Read(reader, binary.BigEndian, &v.signature)
	binary.Read(reader, binary.BigEndian, &v.version)
	binary.Read(reader, binary.BigEndian, &v.flags)
	binary.Read(reader, binary.BigEndian, &v.data_offset)
	fmt.Println(v.sig())
	return v
}

func next_previous_tag_size(reader io.Reader) (uint32, error) {
	var v uint32
	err := binary.Read(reader, binary.BigEndian, &v)
	fmt.Println(v, `prev-tag-size`)
	return v, err
}

const (
	tag_type_audio  byte = 8
	tag_type_video  byte = 9
	tag_type_script byte = 18
)

type uint24 [3]byte
type flv_tag_header struct {
	tag_type  byte   // 8:audio, 9: video, 18: script
	data_size uint24 // data size
	timestamp uint32
	stream_id uint24
}

func (this uint24) to_uint32() uint32 {
	// little-endian
	return uint32(this[0])<<16 | uint32(this[1])<<8 | uint32(this[2])<<0
}

func next_tag_header(reader io.Reader) (flv_tag_header, error) {
	var v flv_tag_header
	err := binary.Read(reader, binary.BigEndian, &v.tag_type)

	if err != nil {
		return v, err
	}
	err = binary.Read(reader, binary.BigEndian, &v.data_size)
	var ds uint24
	if err != nil {
		return v, err
	}
	err = binary.Read(reader, binary.BigEndian, &ds)
	var te byte
	if err != nil {
		return v, err
	}
	err = binary.Read(reader, binary.BigEndian, &te)
	v.timestamp = ds.to_uint32() | (uint32(te) << 24)

	if err != nil {
		return v, err
	}
	err = binary.Read(reader, binary.BigEndian, &v.stream_id)
	fmt.Println(v.tag_type, v.data_size.to_uint32(), v.timestamp, v.stream_id.to_uint32(), `tag-header`)
	return v, err
}

const (
	_ = iota // audio codec id
	sound_format_adpcm
	sound_format_mp3
	sound_format_lpcm
	sound_format_n16kmono
	sound_format_n8kmono
	sound_format_nellymoser
	sound_format_g711apcm
	sound_format_g711mupcm
	_
	sound_format_aac
	sound_format_speex
	_
	_
	sound_format_mp38k
	sound_format_dss
)
const (
	sound_rate_5K = iota
	sound_rate_11K
	sound_rate_22K
	sound_rate_44K
)
const (
	sound_size_8bit = iota
	sound_size_16bit
)
const (
	sound_type_mono = iota
	sound_type_stereo
)

type flv_audio_data_header struct {
	sound_format byte //bit[4]
	sound_rate   byte //bit[2]
	sound_size   byte //bit[1]
	sound_type   byte //bit[1]
	// sound_data []byte
}

const (
	aac_packet_type_aac = iota
	aac_packet_type_raw
)

type aac_audio_data_header struct {
	aac_packet_type byte // aac_packet_type
	// data []byte  0 : AudioSpecConfig same as mp4's esds, 1: raw aac frame data
}

const (
	_                      = iota
	frame_type_keyframe    // for avc
	frame_type_interframe  // for avc
	frame_type_dinterframe // for h.263
	frame_type_gkeyframe   //generated key frame
	frame_type_videoinfo   // command frame
)

const (
	_ = iota
	codec_id_jpeg
	codec_id_h263
	codec_id_screenv
	codec_id_on2vp6
	codec_id_on2vp6a
	codec_id_screenv2
	codec_id_avc
)

type flv_video_data_header struct {
	frame_type byte //bit[4]
	codec_id   byte //bit[4]
	// data []byte
}
