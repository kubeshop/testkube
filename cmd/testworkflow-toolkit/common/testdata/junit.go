package testdata

var BasicJUnit = `<?xml version="1.0" encoding="UTF-8"?>
<!--
This is a basic JUnit-style XML example to highlight the basis structure.

Example by Testmo. Copyright 2023 Testmo GmbH. All rights reserved.
Testmo test management software - https://www.testmo.com/
-->
<testsuites time="15.682687">
    <testsuite name="Tests.Registration" time="6.605871">
        <testcase name="testCase1" classname="Tests.Registration" time="2.113871" />
        <testcase name="testCase2" classname="Tests.Registration" time="1.051" />
        <testcase name="testCase3" classname="Tests.Registration" time="3.441" />
    </testsuite>
    <testsuite name="Tests.Authentication" time="9.076816">
        <testsuite name="Tests.Authentication.Login" time="4.356">
            <testcase name="testCase4" classname="Tests.Authentication.Login" time="2.244" />
            <testcase name="testCase5" classname="Tests.Authentication.Login" time="0.781" />
            <testcase name="testCase6" classname="Tests.Authentication.Login" time="1.331" />
        </testsuite>
        <testcase name="testCase7" classname="Tests.Authentication" time="2.508" />
        <testcase name="testCase8" classname="Tests.Authentication" time="1.230816" />
        <testcase name="testCase9" classname="Tests.Authentication" time="0.982">
            <failure message="Assertion error message" type="AssertionError">
                <!-- Call stack printed here -->
            </failure>
        </testcase>
    </testsuite>
</testsuites>`

var CompleteJUnit = `<?xml version="1.0" encoding="UTF-8"?>
<!--
This is a JUnit-style XML example with commonly used tags and attributes.

Example by Testmo. Copyright 2023 Testmo GmbH. All rights reserved.
Testmo test management software - https://www.testmo.com/
-->

<!-- <testsuites> Usually the root element of a JUnit XML file. Some tools leave out
the <testsuites> element if there is only a single top-level <testsuite> element (which
is then used as the root element).

name        Name of the entire test run
tests       Total number of tests in this file
failures    Total number of failed tests in this file
errors      Total number of errored tests in this file
skipped     Total number of skipped tests in this file
assertions  Total number of assertions for all tests in this file
time        Aggregated time of all tests in this file in seconds
timestamp   Date and time of when the test run was executed (in ISO 8601 format)
-->
<testsuites name="Test run" tests="8" failures="1" errors="1" skipped="1"
    assertions="20" time="16.082687" timestamp="2021-04-02T15:48:23">

    <!-- <testsuite> A test suite usually represents a class, folder or group of tests.
    There can be many test suites in an XML file, and there can be test suites under other
    test suites.

    name        Name of the test suite (e.g. class name or folder name)
    tests       Total number of tests in this suite
    failures    Total number of failed tests in this suite
    errors      Total number of errored tests in this suite
    skipped     Total number of skipped tests in this suite
    assertions  Total number of assertions for all tests in this suite
    time        Aggregated time of all tests in this file in seconds
    timestamp   Date and time of when the test suite was executed (in ISO 8601 format)
    file        Source code file of this test suite
    -->
    <testsuite name="Tests.Registration" tests="8" failures="1" errors="1" skipped="1"
        assertions="20" time="16.082687" timestamp="2021-04-02T15:48:23"
        file="tests/registration.code">

        <!-- <properties> Test suites (and test cases, see below) can have additional
        properties such as environment variables or version numbers. -->
        <properties>
            <!-- <property> Each property has a name and value. Some tools also support
            properties with text values instead of value attributes. -->
            <property name="version" value="1.774" />
            <property name="commit" value="ef7bebf" />
            <property name="browser" value="Google Chrome" />
            <property name="ci" value="https://github.com/actions/runs/1234" />
            <property name="config">
                Config line #1
                Config line #2
                Config line #3
            </property>
        </properties>

        <!-- <system-out> Optionally data written to standard out for the suite.
        Also supported on a test case level, see below. -->
        <system-out>Data written to standard out.</system-out>

        <!-- <system-err> Optionally data written to standard error for the suite.
        Also supported on a test case level, see below. -->
        <system-err>Data written to standard error.</system-err>

        <!-- <testcase> There are one or more test cases in a test suite. A test passed
        if there isn't an additional result element (skipped, failure, error).

        name        The name of this test case, often the method name
        classname   The name of the parent class/folder, often the same as the suite's name
        assertions  Number of assertions checked during test case execution
        time        Execution time of the test in seconds
        file        Source code file of this test case
        line        Source code line number of the start of this test case
        -->
        <testcase name="testCase1" classname="Tests.Registration" assertions="2"
            time="2.436" file="tests/registration.code" line="24" />
        <testcase name="testCase2" classname="Tests.Registration" assertions="6"
            time="1.534" file="tests/registration.code" line="62" />
        <testcase name="testCase3" classname="Tests.Registration" assertions="3"
            time="0.822" file="tests/registration.code" line="102" />

        <!-- Example of a test case that was skipped -->
        <testcase name="testCase4" classname="Tests.Registration" assertions="0"
            time="0" file="tests/registration.code" line="164">
            <!-- <skipped> Indicates that the test was not executed. Can have an optional
            message describing why the test was skipped. -->
            <skipped message="Test was skipped." />
        </testcase>

        <!-- Example of a test case that failed. -->
        <testcase name="testCase5" classname="Tests.Registration" assertions="2"
            time="2.902412" file="tests/registration.code" line="202">
            <!-- <failure> The test failed because one of the assertions/checks failed.
            Can have a message and failure type, often the assertion type or class. The text
            content of the element often includes the failure description or stack trace. -->
            <failure message="Expected value did not match." type="AssertionError">
                <!-- Failure description or stack trace -->
            </failure>
        </testcase>

        <!-- Example of a test case that had errors. -->
        <testcase name="testCase6" classname="Tests.Registration" assertions="0"
            time="3.819" file="tests/registration.code" line="235">
            <!-- <error> The test had an unexpected error during execution. Can have a
            message and error type, often the exception type or class. The text
            content of the element often includes the error description or stack trace. -->
            <error message="Division by zero." type="ArithmeticError">
                <!-- Error description or stack trace -->
            </error>
        </testcase>

        <!-- Example of a test case with outputs. -->
        <testcase name="testCase7" classname="Tests.Registration" assertions="3"
            time="2.944" file="tests/registration.code" line="287">
            <!-- <system-out> Optional data written to standard out for the test case. -->
            <system-out>Data written to standard out.</system-out>

            <!-- <system-err> Optional data written to standard error for the test case. -->
            <system-err>Data written to standard error.</system-err>
        </testcase>

        <!-- Example of a test case with properties -->
        <testcase name="testCase8" classname="Tests.Registration" assertions="4"
            time="1.625275" file="tests/registration.code" line="302">
            <!-- <properties> Some tools also support properties for test cases. -->
            <properties>
                <property name="priority" value="high" />
                <property name="language" value="english" />
                <property name="author" value="Adrian" />
                <property name="attachment" value="screenshots/dashboard.png" />
                <property name="attachment" value="screenshots/users.png" />
                <property name="description">
                    This text describes the purpose of this test case and provides
                    an overview of what the test does and how it works.
                </property>
            </properties>
        </testcase>
    </testsuite>
</testsuites>`

var InvalidJUnit = `<?xml version="1.0" encoding="UTF-8"?>
<!-- This is an invalid JUnit-style XML example to highlight the basis structure. -->
<foo>
	<bar>
</foo>`

var OneLineJUnit = `<?xml version="1.0" encoding="UTF-8"?><testsuites><testsuite name="TestSuite" tests="2" errors="0" failures="1" skipped="0"><testcase name="Test1" classname="TestClass"><failure message="Test failed">Failure details</failure></testcase><testcase name="Test2" classname="TestClass"/></testsuite></testsuites>`

var TestsuitesOnlyJUnit = `<testsuites id="" name="" tests="2" failures="0" skipped="0" errors="0" time="14.511833">
    <testsuite name="smoke.spec.js" timestamp="2024-10-01T12:48:47.332Z" hostname="chromium" tests="1" failures="0" skipped="0" time="6.259" errors="0">
        <testcase name="Smoke 1 - has title" classname="smoke.spec.js" time="6.259"></testcase>
    </testsuite>
    <testsuite name="smoke2.spec.js" timestamp="2024-10-01T12:48:47.332Z" hostname="chromium" tests="1" failures="0" skipped="0" time="6.657" errors="0">
        <testcase name="Smoke 2 - has title" classname="smoke2.spec.js" time="6.657"></testcase>
    </testsuite>
</testsuites>`

var TestsuiteOnlyJUnit = `<testsuite name="smoke.spec.js" timestamp="2024-10-01T12:48:47.332Z" hostname="chromium" tests="1" failures="0" skipped="0" time="6.259" errors="0"><testcase name="Smoke 1 - has title" classname="smoke.spec.js" time="6.259"></testcase></testsuite>`
