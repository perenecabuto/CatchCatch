package io.perenecabuto.catchcatch.view

import android.app.Activity
import io.perenecabuto.catchcatch.CatchCatch

interface ActivityWithApp {
    val app: CatchCatch
    get() {
        return (this as Activity).application as CatchCatch
    }
}