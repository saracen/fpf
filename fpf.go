package fpf // import "github.com/saracen/fpf"

import (
	"bytes"
	"html/template"
	"io"
	"net/url"

	"github.com/saracen/fpf/attr"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// DefaultIncidentInserter is the default incident inserter used if no other
// incident inserter is provided.
var DefaultIncidentInserter = &GenericIncidentInserter{
	ErrorClass: "error",
	Template:   template.Must(template.New("error").Parse(`<ul class="errors">{{ range . }}<li>{{.}}</li>{{end}}</ul>`)),
}

// GenericIncidentInserter provides a basic strategy for inserting error
// messages into the HTML node tree.
type GenericIncidentInserter struct {
	ErrorClass string
	Template   *template.Template
}

// Insert uses a basic strategy for error insertions:
//  • If there is more than one element then error messages are added as
//    children to the elements' lowest common ancestor.
//
//  • If there is only one element, the error messages are inserted beneath it
func (i *GenericIncidentInserter) Insert(elements []LabelableElement, errors []string) error {
	buffer := new(bytes.Buffer)

	// Execute template and pass in errors
	i.Template.Execute(buffer, errors)

	errorNode, err := html.ParseFragment(buffer, &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return err
	}

	addErrorClass := func(node *html.Node) {
		class := attr.Attributes(node.Attr).Attribute("class")
		if class != nil {
			class.Val += " " + i.ErrorClass
		} else {
			node.Attr = append(node.Attr, html.Attribute{Key: "class", Val: i.ErrorClass})
		}
	}

	// Mark elements and labels with error class
	for _, element := range elements {
		addErrorClass(element.Element)
		for _, label := range element.Labels {
			addErrorClass(label)
		}
	}

	switch {
	// Incident is only concerning one input so insert the error underneath it
	case len(elements) == 1:
		elements[0].Element.Parent.InsertBefore(errorNode[0], elements[0].Element.NextSibling)

	// Incident concerns multiple inputs so insert the error as a child to their
	// lowest common ancestor
	default:
		var lca func(a *html.Node, next []LabelableElement) *html.Node
		lca = func(a *html.Node, next []LabelableElement) *html.Node {
			b := next[0].Element
			for ap := a.Parent; ap != nil; ap = ap.Parent {
				for bp := b.Parent; bp != nil; bp = bp.Parent {
					if ap == bp {
						if len(next) > 1 {
							return lca(ap, next[1:])
						}
						return ap
					}
				}
			}
			return a
		}

		ancestor := lca(elements[0].Element, elements[1:])
		ancestor.AppendChild(errorNode[0])
	}

	return nil
}

// IncidentInserter provides an interface for custom error message insertion
// strategies.
//
// The insert method is provided with a list of the affected elements and
// error messages that to be inserted.
type IncidentInserter interface {
	Insert(elements []LabelableElement, errors []string) error
}

type FormPopulationFilter struct {
	// The incident insertion strategy to use
	IncidentInsertion IncidentInserter

	IncludeHiddenInputs   bool // Whether to populate hidden input values
	IncludePasswordInputs bool // Whether to populate password input values
}

// New returns a FormPopulationFilter with default configuration.
func New() *FormPopulationFilter {
	return &FormPopulationFilter{
		IncludeHiddenInputs: true,
	}
}

type processor struct {
	*FormPopulationFilter

	document *html.Node
	labels   []*html.Node
	forms    map[string]*Form
}

// Incident is a collection of one or more form element names and their error
// messages.
//
// Multiple form element names are required when there's a group of elements
// that share common errors. For example, the inputs "new-password" and
// "new-password-confirm" can share the error "passwords do not match".
type Incident struct {
	Names  []string
	Errors []string
}

// LabelableElement contains a form element and its associated labels.
type LabelableElement struct {
	Element *html.Node
	Labels  []*html.Node
}

// Form represents a form by ID that we wish to populate with values and perform
// error insertion on.
type Form struct {
	ID        string
	Values    url.Values
	Incidents []Incident

	// Input elements including:
	// input, button[type="submit"], select, textarea, progress, meter
	inputs []*html.Node

	// Labels associated with an input
	labels map[*html.Node][]*html.Node

	// Options associated with an input
	options map[*html.Node][]*html.Node
}

type formContext struct {
	Form, Label, Select *html.Node
}

func (p *processor) traverse(n *html.Node, context formContext) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "form":
			context.Form = n
		case "label":
			context.Label = n
		}
	}

	// traversal of child nodes when we're done here
	defer func() {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			p.traverse(c, context)
		}
	}()

	if n.Type == html.ElementNode {
		attributes := attr.Attributes(n.Attr)

		// We need to either be in the context of a form or the element has
		// a form attribute for us to find the element interesting.
		formAttribute := attributes.Attribute("form")
		if context.Form == nil && formAttribute == nil {
			return
		}

		// Get form id
		var formId string
		if formAttribute != nil {
			formId = formAttribute.Val
		}
		if formId == "" && context.Form != nil {
			formId = attr.Attributes(context.Form.Attr).Get("id")
		}

		// Are we interested in this form?
		form, ok := p.forms[formId]
		if !ok {
			return
		}

		// Labels can either have a "for" attribute or a "Labelable Element"
		// descendant.
		// We keep a seperate list of those with "for" attributes so we can
		// associate them with their Labelable Element later.
		if n.Data == "label" && attributes.Has("for") {
			p.labels = append(p.labels, n)
			return
		}

		// Is the node an "option" element and in the context of a select?
		if context.Select != nil && n.Data == "option" {
			p.forms[formId].options[context.Select] = append(p.forms[formId].options[context.Select], n)
			return
		}

		// Ignore elements that don't have a "name" attribute, because we
		// can't populate those.
		name := attributes.Get("name")
		if name == "" {
			return
		}

		// Elements we're interested in:
		// input, button[type="submit"], select, textarea, progress, meter
		switch n.Data {
		case "input", "textarea", "progress", "meter":
		case "button":
			if attributes.Get("type") != "submit" {
				return
			}
		case "select":
			context.Select = n
		default:
			return
		}

		// Add input to form inputs slice
		form.inputs = append(form.inputs, n)

		// Associate the label with the element if we're in the context
		// of a label.
		if context.Label != nil && true {
			form.labels[n] = append(form.labels[n], context.Label)
		}
	}
}

func (p *processor) populate(formId string) {
	for _, input := range p.forms[formId].inputs {
		attributes := attr.Attributes(input.Attr)

		name := attributes.Get("name")
		if params, ok := p.forms[formId].Values[name]; ok {
			switch input.Data {
			case "select":
				if options, ok := p.forms[formId].options[input]; ok {
					for _, option := range options {
						optionAttributes := attr.Attributes(option.Attr)
						optionAttributes.Remove("selected")

						value := optionAttributes.Get("value")
						for _, param := range params {
							if value == param {
								option.Attr = append(option.Attr, html.Attribute{Key: "selected", Val: "selected"})
							}
						}
					}
				}

			case "textarea":
				for c := input.FirstChild; c != nil; c = c.NextSibling {
					input.RemoveChild(c)
				}
				input.AppendChild(&html.Node{
					Type: html.TextNode,
					Data: params[0],
				})

			default:
				typ := attributes.Get("type")
				switch typ {
				case "radio":
					fallthrough

				case "checkbox":
					value := attributes.Attribute("value")
					attributes.Remove("checked")
					if value == nil || value.Val == params[0] {
						input.Attr = append(input.Attr, html.Attribute{Key: "checked", Val: "checked"})
					}

				case "file", "image":
					break

				default:
					if typ == "password" && !p.IncludePasswordInputs {
						break
					}
					if typ == "hidden" && !p.IncludeHiddenInputs {
						break
					}
					attributes.Remove("value")
					input.Attr = append(input.Attr, html.Attribute{Key: "value", Val: params[0]})
				}
			}
		}
	}
}

func (p *processor) insert(formId string) error {
	form := p.forms[formId]

	for _, incident := range form.Incidents {
		var elements []LabelableElement

		// An incident can have multiple form element names associated with it.
		// Here we find all of those elements and associated labels to create
		// the LabelableElement.
		for _, input := range form.inputs {
			for _, name := range incident.Names {
				if attr.Attributes(input.Attr).Get("name") != name {
					continue
				}

				elements = append(elements, LabelableElement{
					Element: input,
					Labels:  form.labels[input],
				})
			}
		}

		if len(elements) > 0 {
			if err := p.IncidentInsertion.Insert(elements, incident.Errors); err != nil {
				return err
			}
		}
	}

	return nil
}

// Execute reads from r, modifies forms matching the provided form IDs, and
// writes the output to w. The input is assumed to be UTF-8 encoded.
func (fpf *FormPopulationFilter) Execute(forms []Form, w io.Writer, r io.Reader) error {
	var err error

	p := &processor{FormPopulationFilter: fpf}
	p.forms = make(map[string]*Form)
	for _, form := range forms {
		form.labels = make(map[*html.Node][]*html.Node)
		form.options = make(map[*html.Node][]*html.Node)
		p.forms[form.ID] = &form
	}

	if p.IncidentInsertion == nil {
		p.IncidentInsertion = DefaultIncidentInserter
	}

	p.document, err = html.Parse(r)
	if err != nil {
		return err
	}

	p.traverse(p.document, formContext{})

	// Match labels to associated input elements we were interested in
	for _, form := range p.forms {
		for _, label := range p.labels {
			id := attr.Attributes(label.Attr).Get("for")
			for _, input := range p.forms[form.ID].inputs {
				if id == attr.Attributes(input.Attr).Get("id") {
					p.forms[form.ID].labels[input] = append(p.forms[form.ID].options[input], label)
				}
			}
		}

		// perform value population
		p.populate(form.ID)

		// perform error insertion
		if err = p.insert(form.ID); err != nil {
			return err
		}
	}

	return html.Render(w, p.document)
}

// Execute executes the provided template with the provided data, modifies forms
// matching the provided form IDs, and writes the output to w. The template
// output is assumed to be UTF-8 encoded.
func (fpf *FormPopulationFilter) ExecuteTemplate(forms []Form, w io.Writer, t *template.Template, data interface{}) error {
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return err
	}

	return fpf.Execute(forms, w, buf)
}
