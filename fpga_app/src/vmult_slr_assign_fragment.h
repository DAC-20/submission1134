#ifndef VMULT_SLR_ASSIGN_FRAGMENT_H_
 #define VMULT_SLR_ASSIGN_FRAGMENT_H_

#ifdef __cplusplus
extern "C" {
#endif

int init(char* argv1, char* size, char* max_batch_size);
int compute_batch_all(char* argv2);
int compute_batch_frag0(char* argv2);
int compute_batch_frag1(char* argv2);
int compute_batch_frag2(char* argv2);
void cleanup();

#ifdef __cplusplus
}
#endif

#endif // VMULT_SLR_ASSIGN_FRAGMENT_H_