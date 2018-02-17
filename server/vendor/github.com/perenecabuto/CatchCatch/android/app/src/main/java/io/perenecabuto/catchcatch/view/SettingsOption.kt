package io.perenecabuto.catchcatch.view

import android.annotation.SuppressLint
import android.app.Activity
import android.view.View
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import io.perenecabuto.catchcatch.R

@SuppressLint("ViewConstructor")
class SettingsOption(val context: Activity, val label: String, val onClick: (String) -> Unit) : LinearLayout(context) {

    init {
        View.inflate(context, R.layout.activity_settings_option, this)
    }

    override fun onAttachedToWindow() {
        (findViewById(R.id.activity_settings_option_text) as TextView).text = label
        setOnClickListener {
            Toast.makeText(context, label, Toast.LENGTH_LONG).show()
            onClick(label)
        }
    }
}