package main

import "testing"

func TestParseStudyRequest(t *testing.T) {
	for _, tc := range []struct{ input, command, topic string }{
		{"  Movimentos da Lua  ", "/licao", "Movimentos da Lua"},
		{"/LICAO ciência", "/licao", "ciência"},
		{"/aprofundar   energia  solar", "/aprofundar", "energia  solar"},
		{"/praticar frações", "/praticar", "frações"},
		{"/revisar\tfísica", "/revisar", "física"},
		{"", "", ""}, {"/", "", ""}, {"/licao", "", ""}, {"/terminal calc", "", ""},
	} {
		request, err := parseStudyRequest(tc.input)
		if (err != nil) != (tc.command == "") || request.command != tc.command || request.topic != tc.topic {
			t.Errorf("%q: got %+v, %v", tc.input, request, err)
		}
	}
}

func TestStudyGuidanceUsesRegisteredCommand(t *testing.T) {
	request := studyRequest{command: "/praticar", topic: "Frações"}
	if request.guidance() != studyCommands[2].guidance || request.guidance() == studyCommands[0].guidance {
		t.Fatal("expected trusted practice guidance")
	}
	request.command = "untrusted"
	if request.guidance() != studyCommands[0].guidance {
		t.Fatal("unknown command must not become provider instructions")
	}
}
