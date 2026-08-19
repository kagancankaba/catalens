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
    var result by remember { mutableStateOf<RecognizeResponse?>(null) }
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
                    recognizeImage(bytes)
                }
                result = response
            } catch (e: Exception) {
                Log.e("Catalens", "istek basarisiz", e)
                errorMessage = e.message ?: "bilinmeyen hata"
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
                Text("Kamera")
            }

            Button(
                onClick = onGalleryClick,
                modifier = Modifier.size(width = 140.dp, height = 105.dp),
                shape = RoundedCornerShape(16.dp)
            ) {
                Text("Galeri")
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

@Composable
fun ResultDialog(
    result: RecognizeResponse?,
    errorMessage: String?,
    imageBytes: ByteArray?,
    onDismiss: () -> Unit
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text("Kapat")
            }
        },
        title = {
            Text(if (errorMessage != null) "Hata" else "Sonuç")
        },
        text = {
            Column {
                imageBytes?.let { bytes ->
                    val bitmap = remember(bytes) {
                        BitmapFactory.decodeByteArray(bytes, 0, bytes.size)
                    }
                    bitmap?.let {
                        Image(
                            bitmap = it.asImageBitmap(),
                            contentDescription = null,
                            contentScale = ContentScale.Crop,
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(180.dp)
                                .clip(RoundedCornerShape(12.dp))
                        )
                        Spacer(modifier = Modifier.height(12.dp))
                    }
                }

                errorMessage?.let {
                    Text(it)
                }
                result?.let { response ->
                    response.descriptor?.let { d ->
                        Text("Marka: ${d.brand}")
                        Text("Kategori: ${d.category}")
                        Text("Renk: ${d.colour}")
                        Spacer(modifier = Modifier.height(8.dp))
                    }
                    if (response.noMatch) {
                        Text("Eşleşme bulunamadı")
                    } else {
                        response.matches.forEach { match ->
                            Text("${match.brand} ${match.name} - skor: ${"%.2f".format(match.score)}")
                        }
                    }
                }
            }
        }
    )
}