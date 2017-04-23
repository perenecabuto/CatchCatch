package io.perenecabuto.catchcatch

import android.content.Context
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.os.Handler
import android.util.AttributeSet
import android.util.Log
import android.view.View


class RadarView(context: Context, val attrs: AttributeSet?) : View(context, attrs) {
    val paint = Paint().let { it.style = Paint.Style.STROKE; it }
    val drawHandler: Handler = Handler()

    val steps = 0.5
    var angle = 0.0
    var r: Float = 0.0f
    var center: List<Float> = listOf(0f, 0f)
    val lastPoint = mutableListOf(0f, 0f)

    init {
        autoUpdate()
    }

    override fun onFinishInflate() {
        super.onFinishInflate()
        Log.d("radar center", center.toString() + ":" + r)
    }

    private fun autoUpdate() {
        angle = if (angle >= 360) 0.0 else angle + steps
        lastPoint[0] = center[0] + (r * Math.cos(Math.toRadians(angle))).toFloat()
        lastPoint[1] = center[1] + (r * Math.sin(Math.toRadians(angle))).toFloat()
        invalidate()
        drawHandler.postDelayed(this::autoUpdate, 100)
    }

    override fun onDraw(_canvas: Canvas?) {
        val canvas = _canvas ?: return
        super.onDraw(canvas)
        setupSizes()
        canvas.drawColor(Color.TRANSPARENT)
        canvas.drawCircle(center[0], center[1], r / 3, paint)
        canvas.drawCircle(center[0], center[1], r / 2, paint)
        canvas.drawCircle(center[0], center[1], r / 1.5f, paint)
        canvas.drawCircle(center[0], center[1], r, paint)
        canvas.drawLine(center[0], center[1], lastPoint[0], lastPoint[1], paint)
    }

    private fun setupSizes() {
        if (r == 0f) {
            r = (width / 2).toFloat()
            center = listOf((width / 2).toFloat(), (height / 2).toFloat())
        }
    }
}
