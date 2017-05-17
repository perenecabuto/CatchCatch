package io.perenecabuto.catchcatch.drivers

import android.content.Context
import android.speech.tts.TextToSpeech
import android.speech.tts.TextToSpeech.Engine.KEY_PARAM_UTTERANCE_ID
import android.widget.Toast
import java.io.Serializable
import java.util.*

class GameVoice(context: Context, onComplete: () -> Unit = {}) {
    private var tts: TextToSpeech? = null

    companion object {
        private val voices = mapOf(
            "darkness female" to mapOf("lang" to Locale.UK, "speech_rate" to 0.8F, "pitch" to 0.2F),
            "teen female" to mapOf("lang" to Locale.CANADA, "speech_rate" to 1.2F, "pitch" to 2.1F),
            "light robot female" to mapOf("lang" to Locale.ENGLISH, "speech_rate" to 1F, "pitch" to 0.7F),

            // English male it would be: "en-us-x-sfg#male_1-local"
            // http://stackoverflow.com/questions/9815245/android-text-to-speech-male-voice
            "darkness male" to mapOf("lang" to Locale.UK, "speech_rate" to 0.9F, "pitch" to 0.2F, "voice" to "en-us-x-sfg#male_1-local"),
            "teen male" to mapOf("lang" to Locale.CANADA, "speech_rate" to 1F, "pitch" to 2.1F, "voice" to "en-us-x-sfg#male_1-local"),
            "light robot male" to mapOf("lang" to Locale.ENGLISH, "speech_rate" to 1F, "pitch" to 0.7F, "voice" to "en-us-x-sfg#male_1-local")
        )
        private val default = voices["darkness female"]!!

        fun changeVoice(name: String) {
            voice = voices[name] ?: default
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

    fun speak(msg: String) = tts?.speak(msg, TextToSpeech.QUEUE_FLUSH, null, KEY_PARAM_UTTERANCE_ID)
}
