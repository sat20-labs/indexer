package ord

type MediaType int

const (
	Audio MediaType = iota
	Code
	Font
	Iframe
	Image
	Markdown
	Model
	Pdf
	Text
	Unknown
	Video
)

type MediaInfo struct {
	Type     MediaType
	Property string
	FileExt  []string
}

const (
	ApplicationCbor         = "application/cbor"
	ApplicationJson         = "application/json"
	ApplicationOctetStream  = "application/octet-stream"
	ApplicationPdf          = "application/pdf"
	ApplicationPgpSignature = "application/pgp-signature"
	ApplicationProtobuf     = "application/protobuf"
	ApplicationXJavascript  = "application/x-javascript"
	ApplicationYaml         = "application/yaml"
	AudioFlac               = "audio/flac"
	AudioMpeg               = "audio/mpeg"
	AudioWav                = "audio/wav"
	FontOtf                 = "font/otf"
	FontTtf                 = "font/ttf"
	FontWoff                = "font/woff"
	FontWoff2               = "font/woff2"
	ImageApng               = "image/apng"
	ImageAvif               = "image/avif"
	ImageGif                = "image/gif"
	ImageJpeg               = "image/jpeg"
	ImageJxl                = "image/jxl"
	ImagePng                = "image/png"
	ImageSvgXml             = "image/svg+xml"
	ImageWebp               = "image/webp"
	ModelGltfJson           = "model/gltf+json"
	ModelGltfBinary         = "model/gltf-binary"
	ModelStl                = "model/stl"
	TextCss                 = "text/css"
	TextHtml                = "text/html"
	TextHtmlCharsetUtf8     = "text/html;charset=utf-8"
	TextJavascript          = "text/javascript"
	TextMarkdown            = "text/markdown"
	TextMarkdownUtf8        = "text/markdown;charset=utf-8"
	TextPlain               = "text/plain"
	TextPlainUtf8           = "text/plain;charset=utf-8"
	TextPython              = "text/x-python"
	VideoMp4                = "video/mp4"
	VideoWebm               = "video/webm"
)

var MediaList = map[string]*MediaInfo{
	ApplicationCbor: {
		Type:     Unknown,
		Property: "",
		FileExt:  []string{"cbor"},
	},
	ApplicationJson: {
		Type:     Code,
		Property: "json",
		FileExt:  []string{"json"},
	},
	ApplicationOctetStream: {
		Type:     Unknown,
		Property: "",
		FileExt:  []string{"bin"},
	},
	ApplicationPdf: {
		Type:     Pdf,
		Property: "",
		FileExt:  []string{"pdf"},
	},
	ApplicationPgpSignature: {
		Type:     Text,
		Property: "",
		FileExt:  []string{"asc"},
	},
	ApplicationProtobuf: {
		Type:     Unknown,
		Property: "",
		FileExt:  []string{"binpb"},
	},
	ApplicationXJavascript: {
		Type:     Code,
		Property: "javascript",
		FileExt:  []string{},
	},
	ApplicationYaml: {
		Type:     Code,
		Property: "yaml",
		FileExt:  []string{"yaml", "yml"},
	},
	AudioFlac: {
		Type:     Audio,
		Property: "",
		FileExt:  []string{"flac"},
	},
	AudioMpeg: {
		Type:     Audio,
		Property: "",
		FileExt:  []string{"mp3"},
	},
	AudioWav: {
		Type:     Audio,
		Property: "",
		FileExt:  []string{"wav"},
	},
	FontOtf: {
		Type:     Font,
		Property: "",
		FileExt:  []string{"otf"},
	},
	FontTtf: {
		Type:     Font,
		Property: "",
		FileExt:  []string{"ttf"},
	},
	FontWoff: {
		Type:     Font,
		Property: "",
		FileExt:  []string{"woff"},
	},
	FontWoff2: {
		Type:     Font,
		Property: "",
		FileExt:  []string{"woff2"},
	},
	ImageApng: {
		Type:     Image,
		Property: "pixelated",
		FileExt:  []string{"apng"},
	},
	ImageAvif: {
		Type:     Image,
		Property: "auto",
		FileExt:  []string{"avif"},
	},
	ImageGif: {
		Type:     Image,
		Property: "pixelated",
		FileExt:  []string{"gif"},
	},
	ImageJpeg: {
		Type:     Image,
		Property: "pixelated",
		FileExt:  []string{"jpg", "jpeg"},
	},
	ImageJxl: {
		Type:     Image,
		Property: "auto",
		FileExt:  []string{},
	},
	ImagePng: {
		Type:     Image,
		Property: "pixelated",
		FileExt:  []string{"png"},
	},
	ImageSvgXml: {
		Type:     Iframe,
		Property: "",
		FileExt:  []string{"svg"},
	},
	ImageWebp: {
		Type:     Image,
		Property: "pixelated",
		FileExt:  []string{"webp"},
	},
	ModelGltfJson: {
		Type:     Model,
		Property: "",
		FileExt:  []string{"gltf"},
	},
	ModelGltfBinary: {
		Type:     Model,
		Property: "",
		FileExt:  []string{"glb"},
	},
	ModelStl: {
		Type:     Unknown,
		Property: "",
		FileExt:  []string{"stl"},
	},
	TextCss: {
		Type:     Code,
		Property: "css",
		FileExt:  []string{"css"},
	},
	TextHtml: {
		Type:     Iframe,
		Property: "",
		FileExt:  []string{},
	},
	TextHtmlCharsetUtf8: {
		Type:     Iframe,
		Property: "",
		FileExt:  []string{"html"},
	},
	TextJavascript: {
		Type:     Code,
		Property: "javascript",
		FileExt:  []string{"js"},
	},
	TextMarkdown: {
		Type:     Markdown,
		Property: "",
		FileExt:  []string{},
	},
	TextMarkdownUtf8: {
		Type:     Markdown,
		Property: "",
		FileExt:  []string{"md"},
	},
	TextPlain: {
		Type:     Text,
		Property: "",
		FileExt:  []string{},
	},
	TextPlainUtf8: {
		Type:     Text,
		Property: "",
		FileExt:  []string{"txt"},
	},
	TextPython: {
		Type:     Code,
		Property: "python",
		FileExt:  []string{"py"},
	},
	VideoMp4: {
		Type:     Video,
		Property: "",
		FileExt:  []string{"mp4"},
	},
	VideoWebm: {
		Type:     Video,
		Property: "",
		FileExt:  []string{"webm"},
	},
}

const (
	STRICT_TRANSPORT_SECURITY   = "strict-transport-security"
	CONTENT_SECURITY_POLICY     = "content-security-policy"
	CACHE_CONTROL               = "cache-control"
	VARY                        = "vary"
	ACCESS_CONTROL_ALLOW_ORIGIN = "access-control-allow-origin"
	TRANSFER_ENCODING           = "transfer-encoding"
	CONTENT_ENCODING            = "content-encoding"
)

const (
	CONTEXT_TYPE_TEXT = "text/html; charset=utf-8"
	CONTENT_TYPE_JSON = "application/json"
)

const (
	TXID_LEN              = 64
	MIN_INSCRIPTIONID_LEN = TXID_LEN + 2
)
