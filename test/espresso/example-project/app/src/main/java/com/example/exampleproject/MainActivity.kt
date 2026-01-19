package com.example.exampleproject

import android.os.Bundle
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import android.widget.Button

class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val button: Button = findViewById(R.id.button)
        button.setOnClickListener {
            AlertDialog.Builder(this)
                .setTitle("Dialog")
                .setMessage("Button clicked")
                .setPositiveButton("OK", null)
                .show()
        }
    }
}