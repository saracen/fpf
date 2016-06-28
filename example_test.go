package fpf_test

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	"github.com/saracen/fpf"
)

func ExampleFormPopulationFilter_ExecuteTemplate() {
	const tpl = `
<html>
<head>
	<title>Your Information</title>
</head>
<body>
	<form action="/" method="post">
		<div class="form-group">
			<label for="name">Name</label>
			<input name="name" type="text" placeholder="John Smith" />
		</div>

		<div class="form-group">
			<label>Do you like food? <input type="checkbox" name="food" /></label>
		</div>

		<div class="form-group">
			<label for="month">Favourite Month</label>
			<select id="month" name="month">
			{{- range $index, $month := .Months }}
				<option value="{{ $index }}">{{ $month }}</option>
			{{- end }}
			</select>
		</div>
	</form>
</body>
</html>`

	// Parse template
	t, err := template.New("info").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}

	// Create data we'll use with the template
	var data struct {
		Months []time.Month
	}
	for i := time.January; i <= time.December; i++ {
		data.Months = append(data.Months, i)
	}

	// Set the values we want populated
	values := url.Values{}
	values.Set("name", "Arran Walker")
	values.Set("food", "1")
	values.Set("month", "3")

	// Execute template with fpf
	fp := fpf.New()
	err = fp.ExecuteTemplate([]fpf.Form{{Values: values}}, os.Stdout, t, data)
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	//
	// <html><head>
	// 	<title>Your Information</title>
	// </head>
	// <body>
	// 	<form action="/" method="post">
	// 		<div class="form-group">
	// 			<label for="name">Name</label>
	// 			<input name="name" type="text" placeholder="John Smith" value="Arran Walker"/>
	// 		</div>
	//
	// 		<div class="form-group">
	// 			<label>Do you like food? <input type="checkbox" name="food" checked="checked"/></label>
	// 		</div>
	//
	// 		<div class="form-group">
	// 			<label for="month">Favourite Month</label>
	// 			<select id="month" name="month">
	//				<option value="0">January</option>
	//				<option value="1">February</option>
	//				<option value="2">March</option>
	//				<option value="3" selected="selected">April</option>
	//				<option value="4">May</option>
	//				<option value="5">June</option>
	//				<option value="6">July</option>
	//				<option value="7">August</option>
	//				<option value="8">September</option>
	//				<option value="9">October</option>
	//				<option value="10">November</option>
	//				<option value="11">December</option>
	//			</select>
	//		</div>
	//	</form>
	//
	// </body></html>
}

func ExampleFormPopulationFilter_ExecuteTemplate_errorInsertion() {
	const tpl = `
<html>
<head>
	<title>{{ .Title }}</title>
</head>
<body>
	<form action="/" method="post">
		<div class="form-group">
			<label for="username">Username</label>
			<input name="username" type="text" />
		</div>

		<div class="form-group">
			<div class="form-group-left">
				<label for="password">Password</label>
				<input name="password" type="password" />
			</div>
			<div class="form-group-right">
				<label for="password-confirm">Confirm Password</label>
				<input name="password-confirm" type="password" />
			</div>
		</div>

		<div class="form-group">
			<label>Opt In Newsletter <input type="checkbox" name="newsletter" /></label>
		</div>
		<div class="form-group">
			<label>Opt In Spam <input type="checkbox" name="spam" /></label>
		</div>
	</form>
</body>
</html>`

	// Parse template
	t, err := template.New("register").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}

	// Create validation function
	validate := func(register url.Values) (incidents []fpf.Incident) {
		// validate username
		if len(register.Get("username")) < 5 {
			incidents = append(incidents, fpf.Incident{
				[]string{"username"},
				[]string{"Username needs to be 5 or more characters long."},
			})
		}

		// validate password
		if len(register.Get("password")) < 6 {
			incidents = append(incidents, fpf.Incident{
				[]string{"password", "password-confirm"},
				[]string{"Password needs to be 6 or more characters long."},
			})
		} else if register.Get("password") != register.Get("password-confirm") {
			incidents = append(incidents, fpf.Incident{
				[]string{"password", "password-confirm"},
				[]string{"Passwords do not match."},
			})
		}

		return incidents
	}

	// Using httptest server for example purposes
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var incidents []fpf.Incident

		if r.Method == "POST" {
			err := r.ParseForm()
			if err != nil {
				return
			}

			// Validate posted form
			if incidents = validate(r.PostForm); len(incidents) == 0 {
				// Validated successful
				// Add saving / additional logic
				return
			}
		}

		// Execute template with fpf
		fp := fpf.New()
		fp.ExecuteTemplate([]fpf.Form{{Values: r.PostForm, Incidents: incidents}}, os.Stdout, t, struct{ Title string }{"Registration Page"})
	}))
	defer ts.Close()

	// Emulate browser client post
	resp, err := http.PostForm(ts.URL, url.Values{
		"username":         {"sara"},
		"password":         {"password"},
		"password-comfirm": {"password123"},
		"spam":             {"on"},
	})
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)

	// Output:
	// <html><head>
	// 	<title>Registration Page</title>
	// </head>
	// <body>
	// 	<form action="/" method="post">
	// 		<div class="form-group">
	// 			<label for="username">Username</label>
	// 			<input name="username" type="text" value="sara" class="error"/><ul class="errors"><li>Username needs to be 5 or more characters long.</li></ul>
	// 		</div>
	//
	// 		<div class="form-group">
	// 			<div class="form-group-left">
	// 				<label for="password">Password</label>
	// 				<input name="password" type="password" class="error"/>
	// 			</div>
	// 			<div class="form-group-right">
	// 				<label for="password-confirm">Confirm Password</label>
	// 				<input name="password-confirm" type="password" class="error"/>
	// 			</div>
	// 		<ul class="errors"><li>Passwords do not match.</li></ul></div>
	//
	// 		<div class="form-group">
	// 			<label>Opt In Newsletter <input type="checkbox" name="newsletter"/></label>
	// 		</div>
	// 		<div class="form-group">
	// 			<label>Opt In Spam <input type="checkbox" name="spam" checked="checked"/></label>
	// 		</div>
	// 	</form>
	//
	// </body></html>
}
