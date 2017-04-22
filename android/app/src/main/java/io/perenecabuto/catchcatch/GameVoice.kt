package io.perenecabuto.catchcatch

import android.content.Context
import android.speech.tts.TextToSpeech
import android.widget.Toast
import java.util.*

class GameVoice(context: Context, onComplete: () -> Unit) {
    private var tts: TextToSpeech? = null

    init {
        tts = TextToSpeech(context, finish@ {
            if (it == TextToSpeech.ERROR) {
                Toast.makeText(context, "Failed to start TTS", Toast.LENGTH_LONG).show()
                return@finish
            }
            tts!!.language = Locale.UK
            tts!!.setSpeechRate(0.9F)
            tts!!.setPitch(0.2F)
            onComplete()
        })
    }

    fun speak(msg: String) = tts?.speak(msg, TextToSpeech.QUEUE_FLUSH, null)

}