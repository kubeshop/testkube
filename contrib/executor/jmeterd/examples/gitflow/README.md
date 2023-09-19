# GitFlow Example test for Distributed JMeter

This test is an example of how to run a distributed JMeter test using a git repo as a source and how to use the advanced features of the executor.

## Test Breakdown

### Plugins
All the plugins required by the test are kept in the `plugins` directory of the test folder in the git repo.

### Additional Files
* **CSV**: The test references a CSV file named `Credentials.csv` located in the `data/` directory relative to the project home directory (`${PROJECT_HOME}`). 
  This CSV should contain columns `USERNAME` and `PASSWORD`.

### Environment Variables
* **DATA_CONFIG**: Used to determine the directory of the CSV data file. It defaults to `${PROJECT_HOME}` if not provided.

### Properties
* **JMETER_UC1_NBUSERS**: Number of users for the test. Defaults to `2` if not provided.
* **JMETER_UC1_RAMPUP**: Ramp-up period for the test in seconds. Defaults to `2` if not provided.
* **JMETER_URI_PATH**: The URI path to test against. Defaults to `/pricing` if not provided.

## Steps to execute this Test

1. Push the test folder to your git repo.
2. Set the following environment variable `DATA_CONFIG = /data/repo/<your_test_folder_on_git>`
3. Pass the following arguments when creating test in the Testkube Dashboard `-GJMETER_UC1_NBUSERS=5 jmeter-properties-external.jmx`
4. Create the test as `jmeterd/test` type with `Git` option as per above configuration.
5. Fill the required details like `github repo link`, `username` and `GITHUB_TOKEN`
6. Set the `SLAVES_COUNT` environment variable to the number of slaves you want to spawn for the test.
7. Run the test.
