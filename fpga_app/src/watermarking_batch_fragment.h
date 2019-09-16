#ifndef FRAGMENT_WATERMARKING_BATCH_H_
 #define FRAGMENT_WATERMARKING_BATCH_H_

#ifdef __cplusplus
extern "C" {
#endif

int init(char* argv1, char* argv2);
int compute_batch(char* argv2, char* argv3, char* argv4, char* argv5);
void cleanup();

#ifdef __cplusplus
}
#endif

#endif // FRAGMENT_WATERMARKING_BATCH_H_