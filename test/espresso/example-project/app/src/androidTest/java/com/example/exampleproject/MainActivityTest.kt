package com.example.exampleproject

import androidx.test.ext.junit.rules.ActivityScenarioRule
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.espresso.Espresso.onView
import androidx.test.espresso.action.ViewActions.click
import androidx.test.espresso.matcher.ViewMatchers.withId
import androidx.test.espresso.matcher.ViewMatchers.withText
import androidx.test.espresso.assertion.ViewAssertions.matches
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class MainActivityTest {

    @get:Rule
    val activityRule = ActivityScenarioRule(MainActivity::class.java)

    @Test
    fun dialogIsDisplayedAfterButtonClick() {
        // Check header text is displayed
        onView(withId(R.id.headerText)).check(matches(withText("Example app:")))
        // Click button and check dialog
        onView(withId(R.id.button)).perform(click())
        onView(withText("Button clicked")).check(matches(withText("Button clicked")))
    }
}