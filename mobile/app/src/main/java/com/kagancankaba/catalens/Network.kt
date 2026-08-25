package com.kagancankaba.catalens

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

@Serializable
data class AttributeKV(
    val key: String,
    val value: String
)


@Serializable
data class BoundingBox(
    val xMin: Double,
    val yMin: Double,
    val xMax: Double,
    val yMax: Double
)

@Serializable
data class Descriptor(
    val brand: String,
    val category: String,
    val colour: String,
    val form: String,
    val visibleText: String,
    val attributes: List<AttributeKV>,
    val boundingBox: BoundingBox? = null
)

@Serializable
data class Match(
    val id: String,
    val name: String,
    val brand: String,
    val score: Double
)

@Serializable
data class RecognizeResponse(
    val descriptor: Descriptor? = null,
    val filterApplied: String? = null,
    val matches: List<Match> = emptyList(),
    val noMatch: Boolean = false,
    val substitutes: List<Match> = emptyList()
)

private val client = OkHttpClient()
private val json = Json { ignoreUnknownKeys = true }

fun recognizeImage(imageBytes: ByteArray): RecognizeResponse {
    val requestBody = MultipartBody.Builder()
        .setType(MultipartBody.FORM)
        .addFormDataPart(
            "image",
            "photo.jpg",
            imageBytes.toRequestBody("image/jpeg".toMediaType())
        )
        .build()

    val request = Request.Builder()
        .url("http://127.0.0.1:8080/recognize")
        .post(requestBody)
        .build()

    client.newCall(request).execute().use { response ->
        val body = response.body?.string() ?: throw Exception("empty response")
        return json.decodeFromString<RecognizeResponse>(body)
    }
}

@Serializable
data class ItemResult(
    val descriptor: Descriptor,
    val filterApplied: String? = null,
    val matches: List<Match> = emptyList(),
    val noMatch: Boolean = false
)

@Serializable
data class RecognizeMultiResponse(
    val items: List<ItemResult> = emptyList()
)

fun recognizeImageMulti(imageBytes: ByteArray): RecognizeMultiResponse {
    val requestBody = MultipartBody.Builder()
        .setType(MultipartBody.FORM)
        .addFormDataPart(
            "image",
            "photo.jpg",
            imageBytes.toRequestBody("image/jpeg".toMediaType())
        )
        .build()

    val request = Request.Builder()
        .url("http://127.0.0.1:8080/recognize-multi")
        .post(requestBody)
        .build()

    client.newCall(request).execute().use { response ->
        val body = response.body?.string() ?: throw Exception("empty response")
        return json.decodeFromString<RecognizeMultiResponse>(body)
    }
}