source_filename = "test.ddp"

%garbage_collected = type { %garbage_collected*, i1, i8 }
%ddpstring = type { %garbage_collected, i16*, i64 }

@0 = global [7 x i16] [i16 69, i16 114, i16 102, i16 111, i16 108, i16 103, i16 33]
@1 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 49]
@2 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 50]
@3 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 49]
@4 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 50]

declare external ccc %ddpstring* @inbuilt_string_from_constant(i16* %str, i64 %len)

declare external ccc %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %str)

declare external ccc void @mark_gc(%garbage_collected* %gc)

define i64 @inbuilt_ddpmain() {
0:
	%1 = alloca double
	store double 0x400921FB54411744, double* %1
	%2 = alloca double
	store double 0x4005BF0A8B1407D9, double* %2
	%3 = alloca i64
	store i64 0, i64* %3
	%4 = alloca %ddpstring*
	%5 = alloca %ddpstring*
	br label %6

6:
	%7 = load i64, i64* %3
	%8 = icmp eq i64 %7, 10000
	br i1 %8, label %22, label %12

9:
	%10 = load i64, i64* %3
	%11 = add i64 %10, 1
	store i64 %11, i64* %3
	br label %6

12:
	%13 = bitcast [10 x i16]* @1 to i16*
	%14 = call %ddpstring* @inbuilt_string_from_constant(i16* %13, i64 10)
	%15 = bitcast [10 x i16]* @2 to i16*
	%16 = call %ddpstring* @inbuilt_string_from_constant(i16* %15, i64 10)
	%17 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %14, %ddpstring* %16)
	store %ddpstring* %17, %ddpstring** %4
	%18 = load %ddpstring*, %ddpstring** %4
	%19 = bitcast %ddpstring* %18 to %garbage_collected*
	call void @mark_gc(%garbage_collected* %19)
	%20 = load %ddpstring*, %ddpstring** %4
	%21 = bitcast %ddpstring* %20 to %garbage_collected*
	call void @mark_gc(%garbage_collected* %21)
	br label %9

22:
	%23 = bitcast [10 x i16]* @3 to i16*
	%24 = call %ddpstring* @inbuilt_string_from_constant(i16* %23, i64 10)
	%25 = bitcast [10 x i16]* @4 to i16*
	%26 = call %ddpstring* @inbuilt_string_from_constant(i16* %25, i64 10)
	%27 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %24, %ddpstring* %26)
	store %ddpstring* %27, %ddpstring** %5
	%28 = load %ddpstring*, %ddpstring** %5
	%29 = bitcast %ddpstring* %28 to %garbage_collected*
	call void @mark_gc(%garbage_collected* %29)
	%30 = load %ddpstring*, %ddpstring** %5
	%31 = bitcast %ddpstring* %30 to %garbage_collected*
	call void @mark_gc(%garbage_collected* %31)
	br label %32

32:
	ret i64 0
}

declare external ccc void @inbuilt_Schreibe_Zahl(i64 %p1)

declare external ccc void @inbuilt_Schreibe_Kommazahl(double %p1)

declare external ccc void @inbuilt_Schreibe_Boolean(i1 %p1)

declare external ccc void @inbuilt_Schreibe_Buchstabe(i16 %p1)

declare external ccc void @inbuilt_Schreibe_Text(%ddpstring* %p1)

declare external ccc i64 @inbuilt_Zeit_Seit_Programmstart()

define ccc %ddpstring* @ddpfunc_string_test(%ddpstring* %s1, %ddpstring* %s2) {
0:
	%1 = alloca %ddpstring*
	%2 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %s1)
	store %ddpstring* %2, %ddpstring** %1
	%3 = bitcast %ddpstring* %s1 to %garbage_collected*
	call void @mark_gc(%garbage_collected* %3)
	%4 = alloca %ddpstring*
	%5 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %s2)
	store %ddpstring* %5, %ddpstring** %4
	%6 = bitcast %ddpstring* %s2 to %garbage_collected*
	call void @mark_gc(%garbage_collected* %6)
	%7 = load %ddpstring*, %ddpstring** %1
	call void @inbuilt_Schreibe_Text(%ddpstring* %7)
	call void @inbuilt_Schreibe_Buchstabe(i16 10)
	%8 = load %ddpstring*, %ddpstring** %4
	call void @inbuilt_Schreibe_Text(%ddpstring* %8)
	call void @inbuilt_Schreibe_Buchstabe(i16 10)
	%9 = bitcast [7 x i16]* @0 to i16*
	%10 = call %ddpstring* @inbuilt_string_from_constant(i16* %9, i64 7)
	%11 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %10)
	%12 = bitcast %ddpstring* %10 to %garbage_collected*
	call void @mark_gc(%garbage_collected* %12)
	%13 = load %ddpstring*, %ddpstring** %1
	%14 = bitcast %ddpstring* %13 to %garbage_collected*
	call void @mark_gc(%garbage_collected* %14)
	%15 = load %ddpstring*, %ddpstring** %4
	%16 = bitcast %ddpstring* %15 to %garbage_collected*
	call void @mark_gc(%garbage_collected* %16)
	ret %ddpstring* %11
}
