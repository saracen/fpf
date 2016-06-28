package fpf_test

import (
	"html/template"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/saracen/fpf"
)

func ExampleTemplate() {
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

	t, err := template.New("info").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}

	var data struct {
		Months []time.Month
	}
	for i := time.January; i <= time.December; i++ {
		data.Months = append(data.Months, i)
	}

	values := url.Values{}
	values.Set("name", "Arran Walker")
	values.Set("food", "1")
	values.Set("month", "3")

	form := fpf.Form{Values: values}

	fp := fpf.New()
	err = fp.ExecuteTemplate([]fpf.Form{form}, os.Stdout, t, data)
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
