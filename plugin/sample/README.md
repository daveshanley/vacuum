Hello vacuum user!

If you would like to try out this custom golang plugin, there are a few steps you need to take.

First, make sure you have the code checked out.

```bash
git clone https://github.com/daveshanley/vacuum.git && cd vacuum/plugin/sample 
```

Once checked out, compile the plugin.

```bash
go build -buildmode=plugin boot.go check_single_path.go useless_func.go
```
Go back up into the vacuum directory and compile vacuum

```bash
cd ../../ && go build vacuum.go 
```

Now we can run the sample ruleset that uses custom functions, with an OpenAPI specification. Use the -f flag to specify the path to the sample plugin.

```bash
./vacuum lint -r rulesets/examples/sample-plugin-ruleset.yaml -f plugin/sample /path/to/openapi.yaml
```

vacuum should locate the functions and run without issue. The following output should be displayed:

```bash
 INFO  Located custom function plugin: plugin/sample/sample.so
 INFO  Loaded 2 custom function(s) successfully.
 INFO  Linting against 2 rules: https://quobix.com/vacuum/rulesets/custom-rulesets
```