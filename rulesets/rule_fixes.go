package rulesets

//TODO: look into copy management solutions, because this sucks.
const (
	operationSuccessResponseFix string = "Make sure that your operation returns a 'success' response via  2xx or 3xx " +
		"response code. An API consumer will always expect a success response"

	contactPropertiesFix string = "Complete specification contact information. Fill in the 'name', 'url' and 'email'" +
		"properties so consumers of the spec know how to reach you."

	contactFix string = "The specification 'info' section doesn't contain a 'contact' object. Add it and make sure to complete 'name', 'url' and 'email' " +
		"properties so consumers of the spec know how to reach you."

	infoDescriptionFix string = "The 'info' section is missing a description, surely you want people to know " +
		"what this spec is all about, right?"

	infoLicenseFix string = "The 'info' section is missing a 'license' object. Please add an appropriate one"

	infoLicenseUrlFix string = "The 'info' section license URL is missing. If you add a license, you need to make sure " +
		" that you link to an appropriate URL for that license."

	noEvalInMarkDownFix string = "Remove all references to 'eval()' in the description. These can be used by malicious" +
		" actors to embed code in contracts that is then executed when read by a browser."

	noScriptTagsInMarkdownFix string = "Remove all references to '<script>' tags from the description. These can be used by " +
		"malicious actors to load remote code if the spec is being parsed by a browser."

	openAPITagsAlphabeticalFix string = "The global tags defined in the spec are not listed alphabetically, Everything is " +
		"much better when data is pre-sorted. Order the tags in alphabetical sequence."

	openAPITagsFix string = "Add a global 'tags' object to the root of the spec. Global tags are used by operations to define " +
		"taxonomy and information architecture by tools. Tags generate navigation in documentation as well as modules in code generation."

	oas2APISchemesFix string = "Add an array of supported host 'schemes' to the root of the specification. These are the available " +
		"API schemes (like https/http)."

	oas2HostNotExampleFix string = "Remove 'example.com' from the host URL, it's not going to work."

	oas3HostNotExampleFix string = "Remove 'example.com from the 'servers' URL, it's not going to work."

	oas2HostTrailingSlashFix string = "Remove the trailing slash from the host URL. This may cause some tools to incorrectly " +
		"add a double slash to paths."
)
