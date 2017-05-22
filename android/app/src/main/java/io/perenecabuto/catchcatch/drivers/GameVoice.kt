package io.perenecabuto.catchcatch.drivers

import android.content.Context
import android.content.Context.MODE_PRIVATE
import android.speech.tts.TextToSpeech
import android.speech.tts.TextToSpeech.Engine.KEY_PARAM_UTTERANCE_ID
import android.widget.Toast
import java.io.Serializable
import java.util.*

class GameVoice(val context: Context, onComplete: () -> Unit = {}) {
    private var tts: TextToSpeech? = null
    private val prefs = context.getSharedPreferences(javaClass.simpleName, MODE_PRIVATE)

    internal var voice: String
        get() = prefs.getString("voice", defaultVoice)
        set(value) = prefs.edit().putString("voice", value).apply()

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
        private val defaultVoice = "darkness female"

        fun voices(): Set<String> = voices.keys
    }

    init {
        tts = TextToSpeech(context) finish@ {
            if (it == TextToSpeech.ERROR) {
                Toast.makeText(context, "Failed to start TTS", Toast.LENGTH_LONG).show()
                return@finish
            }

            voices[voice]?.let { setupWithMap(it) }
            onComplete()
        }
    }

    fun speak(msg: String) = tts?.speak(msg, TextToSpeech.QUEUE_ADD, null, KEY_PARAM_UTTERANCE_ID)

    fun changeVoice(name: String): GameVoice {
        voice = name
        voices[voice]?.let { setupWithMap(it) }
        return this
    }

    private fun setupWithMap(settings: Map<String, Serializable>) {
        tts?.let {
            it.language = settings["lang"] as Locale
            it.setSpeechRate(settings["speech_rate"] as Float)
            it.setPitch(settings["pitch"] as Float)
            it.voice = it.voices.firstOrNull { it.name == settings["voice"] as String? } ?: return
            it.engines
        }
    }
}
