package io.perenecabuto.catchcatch

open class BaseDialog(context: android.content.Context) : android.app.Dialog(context) {
    override fun onCreate(savedInstanceState: android.os.Bundle?) {
        super.onCreate(savedInstanceState)
        window.attributes.windowAnimations = io.perenecabuto.catchcatch.R.style.PopUpDialog
        window.setBackgroundDrawableResource(android.R.color.transparent)
    }

    override fun show() {
        try {
            super.show()
        } catch (e: Exception) {
            e.printStackTrace()
            return
        }
    }

    fun showWithTimeout(millis: Long) {
        show()
        android.os.Handler().postDelayed(this::dismiss, millis)
    }
}