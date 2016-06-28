/*
Package fpf provides form value population and error message insertion.

Value Population

Value population populates HTML form elements with provided values.

This is useful when a user is sent back to a form they have previously filled
out, especially in regards to one they submitted but had validation errors.

Value population is achieved by parsing the HTML node tree to discover form
elements. If a value is provided for any discovered form element, then the form
element is populated with the value.

The way the value is populated depends upon the element:

 • textarea: the text content is populated.

 • select: the option matching the value is given the attribute "selected".

 • input[type=radio], input[type=checkbox]: the input is given the "checked"
   attribute.

 • input: the input's "value" attribute is set.

Error Message Insertion

Error message insertion is achieved by providing a list of "incidents". A single
incident can have one or many error messages and also be associated with one or
many form elements.

If a discovered form element has an associated incident, the IncidentInsertion
strategy provided is invoked to insert error messages into the HTML node tree in
relation to the form element and its labels.
*/
package fpf
