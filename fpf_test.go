package fpf

import (
	"bytes"
	"html/template"
	"net/url"
	"strings"
	"testing"
)

type fpfTest struct {
	Input, Want string
	Forms       []Form
	Data        interface{}
}

var tests = []fpfTest{
	// text value population
	{
		`<!DOCTYPE html><html><head></head><body><form action="/"><input type="text" name="foo"></form></body></html>`,
		`<!DOCTYPE html><html><head></head><body><form action="/"><input type="text" name="foo" value="bar"/></form></body></html>`,
		[]Form{
			{Values: url.Values{"foo": []string{"bar"}}},
		},
		nil,
	},

	// checkbox value population
	{
		`<!DOCTYPE html><html><head></head><body><form action="/"><input type="checkbox" name="foo" value="1"></form></body></html>`,
		`<!DOCTYPE html><html><head></head><body><form action="/"><input type="checkbox" name="foo" value="1" checked="checked"/></form></body></html>`,
		[]Form{
			{Values: url.Values{"foo": []string{"1"}}},
		},
		nil,
	},

	// select value population
	{
		`<!DOCTYPE html><html><head></head><body><form action="/"><select name="foo"><option value="bar">bar</option></select></form></body></html>`,
		`<!DOCTYPE html><html><head></head><body><form action="/"><select name="foo"><option value="bar" selected="selected">bar</option></select></form></body></html>`,
		[]Form{
			{Values: url.Values{"foo": []string{"bar"}}},
		},
		nil,
	},

	// select multiple value population
	{
		`<!DOCTYPE html><html><head></head><body><form action="/"><select name="foo"><option value="bar">bar</option><option value="foo">foo</option></select></form></body></html>`,
		`<!DOCTYPE html><html><head></head><body><form action="/"><select name="foo"><option value="bar" selected="selected">bar</option><option value="foo" selected="selected">foo</option></select></form></body></html>`,
		[]Form{
			{Values: url.Values{"foo": []string{"bar", "foo"}}},
		},
		nil,
	},

	// select multiple with optgroups
	{
		`<!DOCTYPE html><html><head></head><body><form action="/"><select name="foo"><optgroup label="opt1"><option value="foo">bar</option></optgroup><optgroup label="opt2"><option value="bar">foo</option></optgroup></select></form></body></html>`,
		`<!DOCTYPE html><html><head></head><body><form action="/"><select name="foo"><optgroup label="opt1"><option value="foo" selected="selected">bar</option></optgroup><optgroup label="opt2"><option value="bar" selected="selected">foo</option></optgroup></select></form></body></html>`,
		[]Form{
			{Values: url.Values{"foo": []string{"bar", "foo"}}},
		},
		nil,
	},

	// textarea population
	{
		`<!DOCTYPE html><html><head></head><body><form action="/"><textarea name="foo">replace</textarea></form></body></html>`,
		`<!DOCTYPE html><html><head></head><body><form action="/"><textarea name="foo">bar</textarea></form></body></html>`,
		[]Form{
			{Values: url.Values{"foo": []string{"bar"}}},
		},
		nil,
	},

	// incident insertion
	{
		`<!DOCTYPE html><html><head></head><body><form action="/"><label for="foo">bar</label><input id="foo" type="text" name="foo"></form></body></html>`,
		`<!DOCTYPE html><html><head></head><body><form action="/"><label for="foo" class="error">bar</label><input id="foo" type="text" name="foo" value="bar" class="error"/><ul class="errors"><li>You&#39;ve stumbled across an error.</li></ul></form></body></html>`,
		[]Form{
			{
				Values: url.Values{"foo": []string{"bar"}},
				Incidents: []Incident{
					{
						[]string{"foo"},
						[]string{"You've stumbled across an error."},
					},
				},
			},
		},
		nil,
	},

	// incident with multiple elements insertion
	{
		`<!DOCTYPE html><html><head></head><body><form action="/"><div class="group"><label for="new-password">bar</label><input id="new-password" type="text" name="new-password"><label for="confirm-password">bar</label><input id="confirm-password" type="text" name="confirm-password"></div></form></body></html>`,
		`<!DOCTYPE html><html><head></head><body><form action="/"><div class="group"><label for="new-password" class="error">bar</label><input id="new-password" type="text" name="new-password" class="error"/><label for="confirm-password" class="error">bar</label><input id="confirm-password" type="text" name="confirm-password" class="error"/><ul class="errors"><li>Passwords did not match.</li></ul></div></form></body></html>`,
		[]Form{
			{
				Values: url.Values{"foo": []string{"bar"}},
				Incidents: []Incident{
					{
						[]string{"new-password", "confirm-password"},
						[]string{"Passwords did not match."},
					},
				},
			},
		},
		nil,
	},
}

var templateTests = []fpfTest{
	// textarea with template class and content
	{
		`<!DOCTYPE html><html><head></head><body><form action="/"><textarea name="foo" class="{{ .Class }}">{{ .Text }}</textarea></form></body></html>`,
		`<!DOCTYPE html><html><head></head><body><form action="/"><textarea name="foo" class="foobar error">bar</textarea><ul class="errors"><li>You&#39;ve stumbled across an error.</li></ul></form></body></html>`,
		[]Form{
			{
				Values: url.Values{"foo": []string{"bar"}},
				Incidents: []Incident{
					{
						[]string{"foo"},
						[]string{"You've stumbled across an error."},
					},
				},
			},
		},
		struct{ Class, Text string }{"foobar", "replace me"},
	},
}

func TestExecute(t *testing.T) {
	for _, test := range tests {
		input := strings.NewReader(test.Input)
		output := new(bytes.Buffer)

		fpf := New()
		fpf.Execute(test.Forms, output, input)

		if string(output.Bytes()) != test.Want {
			t.Errorf("Execute(`%s`):\nGot:\n%s\nExpected:\n%s", test.Input, string(output.Bytes()), test.Want)
		}
	}
}

func TestExecuteTemplate(t *testing.T) {
	for _, test := range templateTests {
		output := new(bytes.Buffer)

		tmpl, err := template.New("test.html").Parse(test.Input)
		if err != nil {
			t.Error(err)
		}

		fpf := New()
		fpf.ExecuteTemplate(test.Forms, output, tmpl, test.Data)

		if string(output.Bytes()) != test.Want {
			t.Errorf("Execute(`%s`):\nGot:\n%s\nExpected:\n%s", test.Input, string(output.Bytes()), test.Want)
		}
	}
}

func TestErrorLocation(t *testing.T) {
	html := `<!DOCTYPE html><html><head></head><body><form action="/"><input type="text" name="foo" /><div><input type="checkbox" name="foo1" /><input type="checkbox" name="foo2" /></div></form></body></html>`
	forms := []Form{
		{
			Values: url.Values{"foo": []string{"bar"}},
			Incidents: []Incident{
				{
					[]string{"foo"},
					[]string{"Error with single element."},
				},
				{
					[]string{"foo1", "foo2"},
					[]string{"Error with multiple elements"},
				},
			},
		},
	}

	ii := &GenericIncidentInserter{
		ErrorClass: "error",
		Template:   DefaultIncidentInserter.Template,
	}
	fpf := New()
	fpf.IncidentInsertion = ii

	wants := map[Location]string{
		Child:  `<!DOCTYPE html><html><head></head><body><form action="/"><input type="text" name="foo" value="bar" class="error"/><div><input type="checkbox" name="foo1" class="error"/><input type="checkbox" name="foo2" class="error"/><ul class="errors"><li>Error with multiple elements</li></ul></div><ul class="errors"><li>Error with single element.</li></ul></form></body></html>`,
		Before: `<!DOCTYPE html><html><head></head><body><form action="/"><ul class="errors"><li>Error with single element.</li></ul><input type="text" name="foo" value="bar" class="error"/><ul class="errors"><li>Error with multiple elements</li></ul><div><input type="checkbox" name="foo1" class="error"/><input type="checkbox" name="foo2" class="error"/></div></form></body></html>`,
		After:  `<!DOCTYPE html><html><head></head><body><form action="/"><input type="text" name="foo" value="bar" class="error"/><ul class="errors"><li>Error with single element.</li></ul><div><input type="checkbox" name="foo1" class="error"/><input type="checkbox" name="foo2" class="error"/></div><ul class="errors"><li>Error with multiple elements</li></ul></form></body></html>`,
	}

	for location, want := range wants {
		ii.SingleElementErrorLocation = location
		ii.MultipleElementErrorLocation = location

		input := strings.NewReader(html)
		output := new(bytes.Buffer)

		err := fpf.Execute(forms, output, input)
		if err != nil {
			t.Error(err)
		}
		if string(output.Bytes()) != want {
			t.Errorf("%s, Execute(`%s`):\nGot:\n%s\nExpected:\n%s", location, html, string(output.Bytes()), want)
		}
	}
}
