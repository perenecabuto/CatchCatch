package io.perenecabuto.catchcatch

import android.annotation.SuppressLint
import android.app.Dialog
import android.content.Context
import android.os.Bundle
import android.widget.TableLayout
import android.widget.TableRow
import android.widget.TextView

class RankDialog(context: Context, val rank: GameRank) : Dialog(context) {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.attributes.windowAnimations = R.style.PopUpDialog
        window.setBackgroundDrawableResource(android.R.color.transparent)
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
    private class PlayerRankRow(context: Context, position: Int, player: String, points: Int) : TableRow(context) {
        init {
            addView(LabelTextView(context, position.toString()))
            addView(LabelTextView(context, player))
            addView(LabelTextView(context, points.toString()))
        }
    }
}
