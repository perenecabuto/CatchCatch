package io.perenecabuto.catchcatch

import android.content.Context
import android.os.Bundle
import android.widget.TextView

class TransparentDialog(context: Context, val msg: String) : BaseDialog(context) {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.attributes.windowAnimations = R.style.PopUpDialog
        window.setBackgroundDrawableResource(android.R.color.transparent)
        setContentView(R.layout.dialog_transparent)

        val container = findViewById(R.id.dialog_transparent_text) as TextView
        container.text = msg
    }
}