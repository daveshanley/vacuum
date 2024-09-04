package rulesets

// TODO: look into copy management solutions, because this sucks.
const (
	operationSuccessResponseFix string = "Make sure that your operation returns a 'success' response via  2xx or 3xx " +
		"response code. An API consumer will always expect a success response"

	contactPropertiesFix string = "Complete specification contact information. Fill in the 'name', 'url' and 'email'" +
		"properties so consumers of the spec know how to reach you."

	contactFix string = "The specification 'info' section doesn't contain a 'contact' object. Add it and make sure to complete 'name', 'url' and 'email' " +
		"properties so consumers of the spec know how to reach you."

	infoDescriptionFix string = "The 'info' section is missing a description, surely you want people to know " +
		"what this spec is all about, right?"

	infoLicenseSPDXFix string = "A license can contain either a URL or an SPDX identifier, but not both, They are mutually exclusive and cannot both be present. Choose one or the other"

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

	oas3HostTrailingSlashFix string = "Remove the trailing slash from the server URL. This may cause some tools to incorrectly " +
		"add a double slash to paths."

	operationDescriptionFix string = "All operations must have a description. Descriptions explain how the operation " +
		"works, and how users should use it and what to expect. Operation descriptions make up the bulk of API documentation." +
		" so please, add a description!"

	oasParameterDescriptionFix = "All parameters should have a description. Descriptions are critical to understanding " +
		"how an API works correctly. Please add a description to all parameters."

	descriptionDuplicationFix string = "Descriptions are only useful, if they are meaningful. If a description is meaningful, " +
		"then it won't be something you copy and paste. Please don't duplicate descriptions, make them deliberate and meaningful."

	componentDescriptionFix string = "Components are the inputs and outputs of a specification. A user needs to be able to " +
		"understand each component and what id does. Descriptions are critical to understanding components. Add a description!"

	oasServersFix string = "Ensure server URIs are correct and valid, check the schemes, ensure descriptions are complete."

	operationIdValidInUrlFix string = "An operationId is critical to correct code generation and operation identification. The operationId " +
		"should really be designed in a way to make it friendly when used as part of a URL. Remove non-standard URL characters."

	operationTagsFix string = "Operations use tags to define the domain(s) they are apart of. Generally a single tag per operation is " +
		"used, however some tools use multiple tags. The point is that you need tags! Add some tags to the operation that match the " +
		"globally available ones."

	pathDeclarationsMustExistFix string = "Paths define the endpoint for operations. Without paths, there is no API. You need to " +
		"add 'paths' to the root of the specification."

	pathNoTrailingSlashFix string = "Paths should not end with a trailing slash, it can confuse tooling and isn't valid as a path " +
		"Remove the trailing slash from the path."

	pathNotIncludeQueryFix string = "Query strings are defined as parameters for an operation, they should not be included in the path " +
		"Please remove it and correctly define as a parameter."

	tagDescriptionRequiredFix string = "Tags are used to group operations into meaningful domains. Without a description, how is anyone " +
		"supposed to understand what the grouping means? Add a description to your global tag."

	typedEnumFix string = "Enum values lock down the number of variable inputs a parameter or schema can have. The problem here is " +
		"that the Enum defined, does not match the specified type. Fix the type!"

	pathParamsFix string = "Path parameters need to match up with the parameters defined for the path, or in an operation " +
		"that sits under that path. Make sure variable names match up and are defined correctly."

	globalOperationTagsFix string = "This tag has not been defined in the global scope, you should always ensure that any tags used" +
		" in operations, are defined globally in the root 'tags' definition."

	operationParametersFix string = "Make sure that all the operation parameters are unique and non-repeating, don't duplicate names, don't" +
		"re-use parameter names in the same operation."

	formDataConsumesFix string = "When using 'formData', the parameter must include the correct mime-types. Make sure you use " +
		"'application/x-www-form-urlencoded' or 'multipart/form-data' as the 'consumes' value in your parameter."

	oas2AnyOfFix string = "You can't use 'anyOf' in Swagger/OpenAPI 2 specs. It was added in version 3. You have to remove it"

	oas2OneOfFix string = "You can't use 'oneOf' in Swagger/OpenAPI 2 specs. It was added in version 3. You have to remove it"

	oas2SchemaCheckFix string = "The schema isn't valid Swagger/OpenAPI 2. Check the errors for more details"

	oas3SchemaCheckFix string = "The schema isn't valid OpenAPI 3. Check the errors for more details"

	operationIdUniqueFix string = "An operationId needs to be unique, there can't be any duplicates in the document, you can't re-use them. " +
		"Make sure the ID used for this operation is unique."

	operationSingleTagFix string = "Using tags as 'groups' for operations makes a lot of sense. It stops making sense when " +
		"multiple tags are used for an operation. Reduce tag count down to one for the operation."

	oas2APIHostFix string = "The 'host' value is missing. How is a user supposed to know where the API actually lives? " +
		"The host is critical in order for consumers to be able to call the API. Add an API host!"

	operationIdExistsFix string = "Every single operation needs an operationId. It's a critical requirement to " +
		"be able to identify each individual operation uniquely. Please add an operationId to the operation."

	duplicatedEntryInEnumFix string = "Enums need to be unique, you can't duplicate them in the same definition. Please remove" +
		" the duplicate value."

	noRefSiblingsFix string = "$ref values must not be placed next to sibling nodes, There should only be a single node " +
		" when using $ref. A common mistake is adding 'description' next to a $ref. This is wrong. remove all siblings!"

	oas3UnusedComponentFix string = "Orphaned components are not used by anything. You might have plans to use them later, " +
		"or they could be older schemas that never got cleaned up. A clean spec is a happy spec. Prune your orphaned components."

	oas3SecurityDefinedFix string = "When defining security values for operations, you need to ensure they match the globally " +
		"defined security schemes. Check $.components.securitySchemes to make sure your values align."

	oas2SecurityDefinedFix string = "When defining security definitions for operations, you need to ensure they match the globally " +
		"defined security schemes. Check $.securityDefinitions to make sure your values align."

	oas2DiscriminatorFix string = "When using polymorphism, a discriminator should also be provided to allow tools to " +
		"understand how to compose your models when generating code. Add a correct discriminator."

	oas3ExamplesFix string = "Examples are critical for consumers to be able to understand schemas and models defined by the spec. " +
		"Without examples, developers can't understand the type of data the API will return in real life. Examples are turned into mocks " +
		"and can provide a rich testing capability for APIs. Add detailed examples everywhere!"

	unusedComponentFix string = "Unused components / definitions are generally the result of the OpenAPI contract being updated without " +
		"considering references. Another reference could have been updated, or an operation changed that no longer references this component. " +
		"Remove this component from the spec, or re-link to it from another component or operation to fix the problem."

	ambiguousPathsFix string = "Paths must all resolve unambiguously, they can't be confused with one another (/{id}/ambiguous and /ambiguous/{id} " +
		"are the same thing. Make sure every path and the variables used are unique and do conflict with one another. Check the ordering of variables " +
		"and the naming of path segments."

	noVerbsInPathFix string = "When HTTP verbs (get/post/put etc) are used in path segments, it muddies the semantics of REST and creates a confusing and " +
		"inconsistent experience. It's highly recommended that verbs are not used in path segments. Replace those HTTP verbs with more meaningful nouns."

	pathsKebabCaseFix string = "Path segments should not contain any uppercase letters, punctuation or underscores. The only valid way to separate words in a " +
		"segment, is to use a hyphen '-'. The elements that are violating the rule are highlighted in the violation description. These are the elements that need to " +
		"change."

	operationsErrorResponseFix string = "Make sure each operation defines at least one 4xx error response. 4xx Errors are " +
		"used to inform clients they are using the API incorrectly, with bad input, or malformed requests. An API with no errors" +
		"defined is really hard to navigate."

	schemaTypeFix string = "Make sure each schema has a value type defined. Without a type, the schema is useless"

	postSuccessResponseFix string = "Make sure your POST operations return a 'success' response via 2xx or 3xx response code. "
)

const (
	owaspNoNumericIDsFix            = "For any parameter which ends in id, use type string with uuid format instead of type integer."
	owaspNoHttpBasicFix             = "Do not use basic authentication method, use a more secure authentication method (e.g., bearer)."
	owaspNoAPIKeysInURLFix          = "Make sure that the apiKey is not part of the URL (path or query): https://blog.stoplight.io/api-keys-best-practices-to-authenticate-apis"
	owaspNoCredentialsInURLFix      = "Remove credentials from the URL."
	owaspAuthInsecureSchemesFix     = "Use a different authorization scheme. Refer to https://www.iana.org/assignments/http-authschemes/ to know more about HTTP Authentication Schemes."
	owaspJWTBestPracticesFix        = "Explicitly state, in the description of the security schemes, that it allows for support of the RFC8725: https://datatracker.ietf.org/doc/html/rfc8725."
	owaspRateLimitFix               = "Implement rate-limiting using HTTP headers: https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/ Customer headers like X-Rate-Limit-Limit (Twitter: https://developer.twitter.com/en/docs/twitter-api/rate-limits) or X-RateLimit-Limit (GitHub: https://docs.github.com/en/rest/overview/resources-in-the-rest-api)"
	owaspRateLimitRetryAfterFix     = "Set the Retry-After header in the 429 response."
	owaspProtectionFix              = "Make sure that all operations should be protected especially when they are not safe (methods that do not alter the state of the server) HTTP methods like `POST`, `PUT`, `PATCH`, and `DELETE`. This is done with one or more non-empty `security` rules. Security rules are defined in the `securityScheme` section."
	owaspDefineErrorValidationFix   = "Extend the responses of all endpoints to support either 400, 422, or 4XX error codes."
	owaspDefineErrorResponses401Fix = "For all endpoints, make sure that the 401 response code is defined as well as its contents."
	owaspDefineErrorResponses500Fix = "For all endpoints, make sure that the 500 response code is defined as well as its contents."
	owaspDefineErrorResponses429Fix = "For all endpoints, make sure that the 429 response code is defined as well as its contents."
	owaspArrayLimitFix              = "Add `maxItems` for Schema of type 'array'. You should ensure that the subschema in `items` is constrained too."
	owaspStringLimitFix             = "Use `maxLength`, `enum`, or `const`."
	owaspStringRestrictedFix        = "Ensure that strings have either a `format`, RegEx `pattern`, `enum`, or `const`."
	owaspIntegerLimitFix            = "Use `minimum` and `maximum` properties for integer types: avoiding negative numbers when positive are expected, or reducing unreasonable iterations like doing something 1000 times when 10 is expected."
	owaspIntegerFormatFix           = "Specify whether int32 or int64 is expected via `format`."
	owaspNoAdditionalPropertiesFix  = "Disable additional properties by setting `additionalProperties` to `false` or add `maxProperties`."
	owaspSecurityHostsHttpsOAS2Fix  = "Ensure that you are using the HTTPS protocol. Learn more about the importance of TLS (over SSL) here: https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Protection_Cheat_Sheet.html."
	owaspSecurityHostsHttpsOAS3Fix  = "Prefix server URLs with the HTTPS protocol: `https://`. Learn more about the importance of TLS (over SSL) here: https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Protection_Cheat_Sheet.html."
)
