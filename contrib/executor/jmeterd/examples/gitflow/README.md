## Steps to execute this Test

1. Push all the files and directories in a github repo in a test folder
2. Set an env variable `DATA_CONFIG = /data/repo/<your_test_folder_on_git>`
3. While creating test in the testkube dashboard pass the args `-GJMETER_UC1_NBUSERS=5 jmeter-properties-external.jmx`
4. Create the test as `jmeterd/test` type with `Git` option as per above configuration.
5. Fill the required details like `github repo link`, `username` and `GITHUB_TOKEN`
5. Add your desired number of slave pods by setting the env `SLAVES_COUNT`.
6. Run the test.
