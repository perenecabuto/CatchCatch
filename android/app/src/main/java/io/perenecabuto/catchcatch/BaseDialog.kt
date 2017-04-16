package io.perenecabuto.catchcatch

open class BaseDialog(context: android.content.Context) : android.app.Dialog(context) {
    override fun onCreate(savedInstanceState: android.os.Bundle?) {
        super.onCreate(savedInstanceState)
        window.attributes.windowAnimations = io.perenecabuto.catchcatch.R.style.PopUpDialog
        window.setBackgroundDrawableResource(android.R.color.transparent)
    }

    fun showWithTimeout(millis: Long) {
        show()
        android.os.Handler().postDelayed(this::dismiss, millis)
    }
}