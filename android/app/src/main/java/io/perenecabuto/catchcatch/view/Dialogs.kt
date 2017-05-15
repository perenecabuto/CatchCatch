package io.perenecabuto.catchcatch.view

import android.annotation.SuppressLint
import android.app.Activity
import android.content.Context
import android.os.Bundle
import android.widget.TableLayout
import android.widget.TableRow
import android.widget.TextView
import io.perenecabuto.catchcatch.model.GameRank
import io.perenecabuto.catchcatch.model.Player
import io.perenecabuto.catchcatch.R

open class BaseDialog(val activity: Activity) : android.app.Dialog(activity) {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.attributes.windowAnimations = R.style.PopUpDialog
        window.setBackgroundDrawableResource(android.R.color.transparent)
    }

    override fun show() {
        if (activity.isDestroyed || activity.isFinishing) return
        try {
            super.show()
        } catch (e: Throwable) {
            e.printStackTrace()
        }
    }

    fun showWithTimeout(millis: Long) {
        show()
        android.os.Handler().postDelayed(finish@ {
            if (activity.isDestroyed || activity.isFinishing) return@finish
            dismiss()
        }, millis)
    }
}

class TransparentDialog(activity: Activity, val msg: String) : BaseDialog(activity) {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.dialog_transparent)

        val container = findViewById(R.id.dialog_transparent_text) as TextView
        container.text = msg
    }
}

class RankDialog(activity: Activity, val rank: GameRank, val you: Player) : BaseDialog(activity) {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.dialog_rank)

        val gameLabel = findViewById(R.id.dialog_rank_game) as TextView
        gameLabel.text = rank.game

        val rankTable = findViewById(R.id.dialog_rank_points) as TableLayout
        rank.pointsPerPlayer.forEachIndexed { i, (player, points) ->
            rankTable.addView(PlayerRankRow(context, i + 1, player, points))
        }
    }

    @SuppressLint("ViewConstructor")
    private class LabelTextView(context: Context, text: String) : TextView(context) {
        init {
            this.text = text
        }
    }

    @SuppressLint("ViewConstructor")
    private inner class PlayerRankRow(context: Context, position: Int, playerID: String, points: Int) : TableRow(context) {
        init {
            addView(LabelTextView(context, position.toString()))
            addView(LabelTextView(context, if (you.id == playerID) "you" else playerID))
            addView(LabelTextView(context, points.toString()))
        }
    }
}