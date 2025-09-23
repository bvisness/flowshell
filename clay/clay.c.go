package clay

/*
#define CLAY_IMPLEMENTATION
#include "clay.h"

Clay_SizingAxis Clay_SetSizingAxisMinMax(Clay_SizingAxis in, Clay_SizingMinMax minMax) {
	in.size.minMax = minMax;
	return in;
}

Clay_SizingAxis Clay_SetSizingAxisPercent(Clay_SizingAxis in, float percent) {
	in.size.percent = percent;
	return in;
}

void Clay_SetContextErrorHandler(Clay_Context* ctx, Clay_ErrorHandler eh) {
	ctx->errorHandler = eh;
}

Clay_RectangleRenderData Clay_GetRenderDataRectangle(Clay_RenderData d) {
	return d.rectangle;
}

Clay_BorderRenderData Clay_GetRenderDataBorder(Clay_RenderData d) {
	return d.border;
}

Clay_TextRenderData Clay_GetRenderDataText(Clay_RenderData d) {
	return d.text;
}

Clay_ImageRenderData Clay_GetRenderDataImage(Clay_RenderData d) {
	return d.image;
}

Clay_ClipRenderData Clay_GetRenderDataClip(Clay_RenderData d) {
	return d.clip;
}

Clay_CustomRenderData Clay_GetRenderDataCustom(Clay_RenderData d) {
	return d.custom;
}

extern void clayErrorCallback(Clay_ErrorData);
extern Clay_Dimensions clayMeasureTextCallback(Clay_StringSlice, Clay_TextElementConfig*, void*);

void clayErrorCallback_cgo(Clay_ErrorData errorText) {
	clayErrorCallback(errorText);
}

Clay_Dimensions clayMeasureTextCallback_cgo(Clay_StringSlice text, Clay_TextElementConfig *config, void *userData) {
	clayMeasureTextCallback(text, config, userData);
}
*/
import "C"
