package config

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name, text string
		ok         bool
	}{{"valid", "workspace = \"1\"\ntoken = \"secret\"\ntimezone = \"UTC\"\n[projects.\"2\"]\nhidden = true\ndisplay_order = 4\n", true}, {"missing", "workspace = \"1\"", false}, {"invalid project", "workspace = \"1\"\ntoken = \"x\"\n[projects.bad]\nhidden = true", false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, e := Parse(tt.text)
			if (e == nil) != tt.ok {
				t.Fatalf("error=%v", e)
			}
			if tt.ok && (!c.Projects[2].Hidden || c.Timezone != "UTC") {
				t.Fatalf("%+v", c)
			}
		})
	}
}
