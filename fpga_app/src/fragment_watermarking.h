#ifndef FRAGMENT_WATERMARKING_H_
 #define FRAGMENT_WATERMARKING_H_

#ifdef __cplusplus
extern "C" {
#endif

int init(char* argv1, char* argv2, char* argv3);
int compute_batch(char* argv5);
void cleanup();

#ifdef __cplusplus
}
#endif

#endif // FRAGMENT_WATERMARKING_H_