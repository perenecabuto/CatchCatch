package io.perenecabuto.catchcatch

import android.annotation.SuppressLint
import android.content.Context
import android.os.Bundle
import android.widget.TableLayout
import android.widget.TableRow
import android.widget.TextView

open class BaseDialog(context: android.content.Context) : android.app.Dialog(context) {
    override fun onCreate(savedInstanceState: android.os.Bundle?) {
        super.onCreate(savedInstanceState)
        window.attributes.windowAnimations = io.perenecabuto.catchcatch.R.style.PopUpDialog
        window.setBackgroundDrawableResource(android.R.color.transparent)
    }

    override fun show() {
        try {
            super.show()
        } catch (e: Throwable) {
            e.printStackTrace()
            return
        }
    }

    fun showWithTimeout(millis: Long) {
        show()
        android.os.Handler().postDelayed(this::dismiss, millis)
    }
}

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

class RankDialog(context: Context, val rank: GameRank) : BaseDialog(context) {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.dialog_rank)

        val gameLabel = findViewById(R.id.dialog_rank_game) as TextView
        gameLabel.text = rank.game

        val rankTable = findViewById(R.id.dialog_rank_points) as TableLayout
        rank.pointsPerPlayer.forEachIndexed { i, (player, points) ->
            rankTable.addView(PlayerRankRow(context, i + 1, player, points))
        }

        findViewById(R.id.dialog_rank).setOnClickListener { dismiss() }
    }

    @SuppressLint("ViewConstructor")
    private class LabelTextView(context: Context, text: String) : TextView(context) {
        init {
            this.text = text
        }
    }

    @SuppressLint("ViewConstructor")
    private class PlayerRankRow(context: Context, position: Int, player: String, points: Int) : TableRow(context) {
        init {
            addView(LabelTextView(context, position.toString()))
            addView(LabelTextView(context, player))
            addView(LabelTextView(context, points.toString()))
        }
    }
}