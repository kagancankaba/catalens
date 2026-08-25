package com.kagancankaba.catalens

import android.Manifest
import android.content.pm.PackageManager
import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageCapture
import androidx.camera.core.ImageCaptureException
import androidx.camera.core.ImageProxy
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import com.kagancankaba.catalens.ui.theme.CatalensTheme
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.util.concurrent.Executors
import androidx.compose.ui.draw.clip
import android.graphics.BitmapFactory
import androidx.compose.foundation.Image
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import android.graphics.Bitmap

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            CatalensTheme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    CatalensApp(modifier = Modifier.padding(innerPadding))
                }
            }
        }
    }
}

private enum class Screen { HOME, CAMERA }

@Composable
fun CatalensApp(modifier: Modifier = Modifier) {
    val context = LocalContext.current
    var hasCameraPermission by remember {
        mutableStateOf(
            ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED
        )
    }

    var screen by remember { mutableStateOf(Screen.HOME) }
    var result by remember { mutableStateOf<RecognizeMultiResponse?>(null) }
    var isLoading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var resultImageBytes by remember { mutableStateOf<ByteArray?>(null) }
    val scope = rememberCoroutineScope()

    fun processImage(bytes: ByteArray) {
        screen = Screen.HOME
        result = null
        errorMessage = null
        resultImageBytes = bytes
        isLoading = true
        scope.launch {
            try {
                val response = withContext(Dispatchers.IO) {
                    recognizeImageMulti(bytes)
                }
                result = response
            } catch (e: Exception) {
                Log.e("Catalens", "request failed", e)
                errorMessage = e.message ?: "unknown error"
            } finally {
                isLoading = false
            }
        }
    }

    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { granted ->
        hasCameraPermission = granted
        if (granted) {
            screen = Screen.CAMERA
        }
    }

    val galleryLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.PickVisualMedia()
    ) { uri ->
        uri?.let {
            val bytes = context.contentResolver.openInputStream(it)?.use { stream -> stream.readBytes() }
            if (bytes != null) {
                processImage(bytes)
            }
        }
    }

    when (screen) {
        Screen.HOME -> {
            HomeScreen(
                modifier = modifier,
                isLoading = isLoading,
                onCameraClick = {
                    if (hasCameraPermission) {
                        screen = Screen.CAMERA
                    } else {
                        permissionLauncher.launch(Manifest.permission.CAMERA)
                    }
                },
                onGalleryClick = {
                    galleryLauncher.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))
                }
            )
        }
        Screen.CAMERA -> {
            CameraPreviewWithCapture(
                modifier = modifier.fillMaxSize(),
                onImageCaptured = { bytes -> processImage(bytes) },
                onCancel = { screen = Screen.HOME }
            )
        }
    }

    if (result != null || errorMessage != null) {
        ResultDialog(
            result = result,
            errorMessage = errorMessage,
            imageBytes = resultImageBytes,
            onDismiss = {
                result = null
                errorMessage = null
                resultImageBytes = null
            }
        )
    }
}

@Composable
fun HomeScreen(
    modifier: Modifier = Modifier,
    isLoading: Boolean,
    onCameraClick: () -> Unit,
    onGalleryClick: () -> Unit
) {
    Column(
        modifier = modifier.fillMaxSize(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Text(
            "CATALENS",
            style = MaterialTheme.typography.headlineMedium,
            color = MaterialTheme.colorScheme.onSurface
        )

        Spacer(modifier = Modifier.height(24.dp))

        Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
            Button(
                onClick = onCameraClick,
                modifier = Modifier.size(width = 140.dp, height = 105.dp),
                shape = RoundedCornerShape(16.dp)
            ) {
                Text("Camera")
            }

            Button(
                onClick = onGalleryClick,
                modifier = Modifier.size(width = 140.dp, height = 105.dp),
                shape = RoundedCornerShape(16.dp)
            ) {
                Text("Gallery")
            }
        }

        Spacer(modifier = Modifier.height(24.dp))

        if (isLoading) {
            CircularProgressIndicator()
        }
    }
}

@Composable
fun CameraPreviewWithCapture(
    modifier: Modifier = Modifier,
    onImageCaptured: (ByteArray) -> Unit,
    onCancel: () -> Unit
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    val previewView = remember { PreviewView(context) }
    val imageCapture = remember { ImageCapture.Builder().build() }
    val executor = remember { Executors.newSingleThreadExecutor() }

    androidx.compose.runtime.LaunchedEffect(Unit) {
        val cameraProvider = withContext(Dispatchers.IO) {
            ProcessCameraProvider.getInstance(context).get()
        }
        val preview = Preview.Builder().build().also {
            it.setSurfaceProvider(previewView.surfaceProvider)
        }
        cameraProvider.unbindAll()
        cameraProvider.bindToLifecycle(
            lifecycleOwner,
            CameraSelector.DEFAULT_BACK_CAMERA,
            preview,
            imageCapture
        )
    }

    Box(modifier = modifier) {
        AndroidView(
            factory = { previewView },
            modifier = Modifier.fillMaxSize()
        )

        Box(
            modifier = Modifier
                .align(Alignment.TopStart)
                .padding(20.dp)
                .size(40.dp)
                .clip(CircleShape)
                .background(Color.Black.copy(alpha = 0.4f))
                .clickable { onCancel() },
            contentAlignment = Alignment.Center
        ) {
            Text("✕", color = Color.White)
        }

        Box(
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .padding(bottom = 32.dp)
                .size(72.dp)
                .clip(CircleShape)
                .background(Color.White.copy(alpha = 0.3f))
                .border(3.dp, Color.White, CircleShape)
                .clickable {
                    imageCapture.takePicture(
                        executor,
                        object : ImageCapture.OnImageCapturedCallback() {
                            override fun onCaptureSuccess(image: ImageProxy) {
                                val buffer = image.planes[0].buffer
                                val bytes = ByteArray(buffer.remaining())
                                buffer.get(bytes)
                                image.close()
                                onImageCaptured(bytes)
                            }

                            override fun onError(exception: ImageCaptureException) {
                                Log.e("Catalens", "capture failed", exception)
                            }
                        }
                    )
                },
            contentAlignment = Alignment.Center
        ) {
            Box(
                modifier = Modifier
                    .size(58.dp)
                    .clip(CircleShape)
                    .background(Color.White)
            )
        }
    }
}

private fun cropToBoundingBox(bitmap: Bitmap, box: BoundingBox?): Bitmap {
    if (box == null) return bitmap
    val left = (box.xMin * bitmap.width).toInt().coerceIn(0, bitmap.width - 1)
    val top = (box.yMin * bitmap.height).toInt().coerceIn(0, bitmap.height - 1)
    val right = (box.xMax * bitmap.width).toInt().coerceIn(left + 1, bitmap.width)
    val bottom = (box.yMax * bitmap.height).toInt().coerceIn(top + 1, bitmap.height)
    return Bitmap.createBitmap(bitmap, left, top, right - left, bottom - top)
}

@Composable
fun ResultDialog(
    result: RecognizeMultiResponse?,
    errorMessage: String?,
    imageBytes: ByteArray?,
    onDismiss: () -> Unit
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text("Close")
            }
        },
        title = {
            Text(if (errorMessage != null) "Error" else "Result")
        },
        text = {
            Column {
                if (errorMessage != null) {
                    Text(errorMessage)
                } else if (result != null && imageBytes != null) {
                    val bitmap = remember(imageBytes) {
                        BitmapFactory.decodeByteArray(imageBytes, 0, imageBytes.size)
                    }
                    val items = result.items

                    if (bitmap != null && items.isNotEmpty()) {
                        val pagerState = rememberPagerState(pageCount = { items.size })

                        HorizontalPager(state = pagerState) { page ->
                            val item = items[page]
                            val cropped = remember(bitmap, item.descriptor.boundingBox) {
                                cropToBoundingBox(bitmap, item.descriptor.boundingBox)
                            }
                            Column {
                                Image(
                                    bitmap = cropped.asImageBitmap(),
                                    contentDescription = null,
                                    contentScale = ContentScale.Crop,
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .height(180.dp)
                                        .clip(RoundedCornerShape(12.dp))
                                )
                                Spacer(modifier = Modifier.height(12.dp))

                                Text("Brand: ${item.descriptor.brand}")
                                Text("Category: ${item.descriptor.category}")
                                Text("Colour: ${item.descriptor.colour}")
                                item.descriptor.attributes.forEach { attr ->
                                    Text("${attr.key}: ${attr.value}")
                                }
                                Spacer(modifier = Modifier.height(8.dp))
                                Text(
                                    if (item.filterApplied != null) {
                                        "Filter applied: ${item.filterApplied}"
                                    } else {
                                        "Filter applied: none (searched all categories)"
                                    }
                                )
                                Spacer(modifier = Modifier.height(8.dp))

                                if (item.noMatch) {
                                    Text("No match found")
                                } else {
                                    item.matches.forEach { match ->
                                        Text("${match.brand} ${match.name} - similarity: ${"%.2f".format(match.score)}")
                                    }
                                }
                            }
                        }

                        if (items.size > 1) {
                            Spacer(modifier = Modifier.height(12.dp))
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.Center
                            ) {
                                repeat(items.size) { index ->
                                    val selected = pagerState.currentPage == index
                                    Box(
                                        modifier = Modifier
                                            .padding(4.dp)
                                            .size(8.dp)
                                            .clip(CircleShape)
                                            .background(
                                                if (selected) MaterialTheme.colorScheme.onSurface
                                                else MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f)
                                            )
                                    )
                                }
                            }
                        }
                    } else {
                        Text("No products detected")
                    }
                }
            }
        }
    )
}