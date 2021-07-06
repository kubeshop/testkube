package newman

type TextResult struct {
	Output []byte
}

type JSONResult struct {
	Collection Collection `json:"Collection"`
	Run        Run        `json:"Run"`
}
type Info struct {
	Name string `json:"Name"`
	ID   string `json:"Id"`
}
type Collection struct {
	Info Info `json:"Info"`
}
type Requests struct {
	Total   int `json:"total"`
	Pending int `json:"pending"`
	Failed  int `json:"failed"`
}
type Assertions struct {
	Total   int `json:"total"`
	Pending int `json:"pending"`
	Failed  int `json:"failed"`
}
type Stats struct {
	Requests   Requests   `json:"Requests"`
	Assertions Assertions `json:"Assertions"`
}
type Timings struct {
	ResponseAverage  int   `json:"responseAverage"`
	ResponseMin      int   `json:"responseMin"`
	ResponseMax      int   `json:"responseMax"`
	ResponseSd       int   `json:"responseSd"`
	DNSAverage       int   `json:"dnsAverage"`
	DNSMin           int   `json:"dnsMin"`
	DNSMax           int   `json:"dnsMax"`
	DNSSd            int   `json:"dnsSd"`
	FirstByteAverage int   `json:"firstByteAverage"`
	FirstByteMin     int   `json:"firstByteMin"`
	FirstByteMax     int   `json:"firstByteMax"`
	FirstByteSd      int   `json:"firstByteSd"`
	Started          int64 `json:"started"`
	Completed        int64 `json:"completed"`
}
type Run struct {
	Stats    Stats         `json:"Stats"`
	Failures []interface{} `json:"Failures"`
	Timings  Timings       `json:"Timings"`
}
