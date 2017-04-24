package io.perenecabuto.catchcatch

import android.content.Context
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.os.Handler
import android.util.AttributeSet
import android.view.View
import java.lang.Math.abs


class RadarView(context: Context, val attrs: AttributeSet?) : View(context, attrs) {
    val paint = Paint().let {
        it.style = Paint.Style.FILL
        it.color = Color.argb(32, 16, 127, 32)
        it.isAntiAlias = true
        it.strokeWidth = 3f
        it
    }

    val drawHandler = Handler()
    val fps = 60L
    val steps = 2f
    var angle = 0.0f
    var r: Float = 0.0f
    var center: List<Float> = listOf(0f, 0f)
    var topOffset: Float = 0.0f

    override fun onLayout(changed: Boolean, left: Int, top: Int, right: Int, bottom: Int) {
        super.onLayout(changed, left, top, right, right)
    }

    override fun onSizeChanged(w: Int, h: Int, oldw: Int, oldh: Int) {
        super.onSizeChanged(w, h, oldw, oldh)
        r = (w / 2).toFloat()
        center = listOf((w / 2).toFloat(), (h / 2).toFloat())
        topOffset = abs(height / 2 - width / 2).toFloat()
    }

    override fun onDraw(canvas: Canvas) {
        super.onDraw(canvas)
        angle = if (angle >= 360) 0.0f else angle + steps
        canvas.drawArc(0f, topOffset, width.toFloat(), width + topOffset, angle, 60f, true, paint)
        canvas.drawCircle(center[0], center[1], r, paint)
        if (!isInEditMode) drawHandler.postDelayed(this::invalidate, 1000 / fps)
    }
}
