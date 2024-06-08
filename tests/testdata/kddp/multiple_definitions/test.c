extern void Not_Mangled(void);
extern long long Not_Mangled_Var;
extern long long Not_Mangled_Var2;

long long Callback(void) {
	Not_Mangled();
	return Not_Mangled_Var + Not_Mangled_Var2;
}
