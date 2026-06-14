package installspec

import "testing"

const zzskel = `{
  "name": "ZZSKEL*1.0*1",
  "source": { "kind": "hfs", "path": "dist/kids/ZZSKEL.kids" },
  "environmentCheck": "run",
  "answers": {
    "rebuildMenuTrees": false,
    "inhibitLogons": false,
    "disableOptionsProtocols": false,
    "delayInstallMinutes": 0
  },
  "device": { "queue": false }
}`

func TestParse_Valid(t *testing.T) {
	s, err := Parse([]byte(zzskel))
	if err != nil {
		t.Fatalf("Parse valid: %v", err)
	}
	if s.Name != "ZZSKEL*1.0*1" || s.Source.Kind != "hfs" {
		t.Errorf("spec = %+v", s)
	}
	// The standard answers map onto the KIDS XPDDIQ answer codes.
	if got := s.Answers.XPDDIQ(); got["XPZ1"] != "0" || got["XPO1"] != "0" || got["XPI1"] != "0" {
		t.Errorf("XPDDIQ = %v, want all 0 (NO)", got)
	}
}

func TestAnswers_XPDDIQ_Yes(t *testing.T) {
	s, _ := Parse([]byte(`{"name":"ZZSKEL*1.0*1","source":{"kind":"hfs","path":"x"},
	  "answers":{"disableOptionsProtocols":true},"device":{"queue":false}}`))
	if s.Answers.XPDDIQ()["XPZ1"] != "1" {
		t.Errorf("disableOptionsProtocols=true must map XPZ1=1")
	}
}

func TestParse_Invalid(t *testing.T) {
	cases := map[string]string{
		"missing name":    `{"source":{"kind":"hfs","path":"x"},"device":{}}`,
		"bad name":        `{"name":"zzskel","source":{"kind":"hfs","path":"x"},"device":{}}`,
		"bad source kind": `{"name":"ZZSKEL*1.0*1","source":{"kind":"ftp","path":"x"},"device":{}}`,
		"no source path":  `{"name":"ZZSKEL*1.0*1","source":{"kind":"hfs"},"device":{}}`,
		"bad envcheck":    `{"name":"ZZSKEL*1.0*1","source":{"kind":"hfs","path":"x"},"environmentCheck":"maybe","device":{}}`,
		"delay too high":  `{"name":"ZZSKEL*1.0*1","source":{"kind":"hfs","path":"x"},"answers":{"delayInstallMinutes":99},"device":{}}`,
		"unknown field":   `{"name":"ZZSKEL*1.0*1","source":{"kind":"hfs","path":"x"},"device":{},"oops":1}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
