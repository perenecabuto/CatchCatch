package io.perenecabuto.catchcatch.drivers

import android.content.Context
import android.speech.tts.TextToSpeech
import android.widget.Toast
import java.util.*

class GameVoice(context: Context, onComplete: () -> Unit) {
    private var tts: TextToSpeech? = null

    companion object {
        private val voices = mapOf(
            "darkness" to mapOf("lang" to Locale.UK, "speech_rate" to 0.9F, "pitch" to 0.2F),
            "teen" to mapOf("lang" to Locale.CANADA, "speech_rate" to 1F, "pitch" to 2.1F),
            "light_robot" to mapOf("lang" to Locale.ENGLISH, "speech_rate" to 1F, "pitch" to 0.7F)
        )
        private var voice = voices["light_robot"]!!

        fun changeVoice(name: String) {
            voice = voices[name] ?: return
        }

        fun voices(): Set<String> {
            return voices.keys
        }
    }

    init {
        tts = TextToSpeech(context, finish@ {
            if (it == TextToSpeech.ERROR) {
                Toast.makeText(context, "Failed to start TTS", Toast.LENGTH_LONG).show()
                return@finish
            }

            tts!!.language = voice["lang"] as Locale
            tts!!.setSpeechRate(voice["speech_rate"] as Float)
            tts!!.setPitch(voice["pitch"] as Float)
            onComplete()
        })
    }

    fun speak(msg: String) = tts?.speak(msg, TextToSpeech.QUEUE_FLUSH, null)
}
