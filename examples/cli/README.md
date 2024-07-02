This example showcases how to use this package to smoke test/end-to-end test(whatever you want to call it)
a CLI application.

We capture the expected output of the CLI application, and compare that to what we get from the code
itself. Using the `snap` package allows for easy automatic updating of the expected values(exit code, stdout, stderr) when they
differ.
