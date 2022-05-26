source_filename = "test.ddp"

%ddpstring = type { i16*, i64 }

@0 = global [7 x i16] [i16 69, i16 114, i16 102, i16 111, i16 108, i16 103, i16 33]
@1 = global [5 x i16] [i16 116, i16 101, i16 115, i16 116, i16 49]
@2 = global [5 x i16] [i16 116, i16 101, i16 115, i16 116, i16 49]
@3 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 49]
@4 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 50]
@5 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 49]
@6 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 50]

declare external ccc %ddpstring* @inbuilt_string_from_constant(i16* %str, i64 %len)

declare external ccc %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %str)

declare external ccc void @inbuilt_decrement_ref_count(i8* %key)

declare external ccc void @inbuilt_increment_ref_count(i8* %key, i8 %kind)

define i64 @inbuilt_ddpmain() {
0:
	%1 = alloca double
	store double 0x400921FB54411744, double* %1
	%2 = alloca double
	store double 0x4005BF0A8B1407D9, double* %2
	%3 = bitcast [5 x i16]* @1 to i16*
	%4 = call %ddpstring* @inbuilt_string_from_constant(i16* %3, i64 5)
	%5 = bitcast [5 x i16]* @2 to i16*
	%6 = call %ddpstring* @inbuilt_string_from_constant(i16* %5, i64 5)
	%7 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %4, %ddpstring* %6)
	call void @inbuilt_Schreibe_Text(%ddpstring* %7)
	%8 = alloca i64
	store i64 0, i64* %8
	%9 = alloca %ddpstring*
	%10 = alloca %ddpstring*
	br label %11

11:
	%12 = load i64, i64* %8
	%13 = icmp eq i64 %12, 10000
	br i1 %13, label %26, label %17

14:
	%15 = load i64, i64* %8
	%16 = add i64 %15, 1
	store i64 %16, i64* %8
	br label %11

17:
	%18 = bitcast [10 x i16]* @3 to i16*
	%19 = call %ddpstring* @inbuilt_string_from_constant(i16* %18, i64 10)
	%20 = bitcast [10 x i16]* @4 to i16*
	%21 = call %ddpstring* @inbuilt_string_from_constant(i16* %20, i64 10)
	%22 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %19, %ddpstring* %21)
	store %ddpstring* %22, %ddpstring** %9
	%23 = bitcast %ddpstring* %22 to i8*
	call void @inbuilt_increment_ref_count(i8* %23, i8 0)
	%24 = load %ddpstring*, %ddpstring** %9
	%25 = bitcast %ddpstring* %24 to i8*
	call void @inbuilt_decrement_ref_count(i8* %25)
	br label %14

26:
	%27 = bitcast [10 x i16]* @5 to i16*
	%28 = call %ddpstring* @inbuilt_string_from_constant(i16* %27, i64 10)
	%29 = bitcast [10 x i16]* @6 to i16*
	%30 = call %ddpstring* @inbuilt_string_from_constant(i16* %29, i64 10)
	%31 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %28, %ddpstring* %30)
	store %ddpstring* %31, %ddpstring** %10
	%32 = bitcast %ddpstring* %31 to i8*
	call void @inbuilt_increment_ref_count(i8* %32, i8 0)
	%33 = load %ddpstring*, %ddpstring** %10
	%34 = bitcast %ddpstring* %33 to i8*
	call void @inbuilt_decrement_ref_count(i8* %34)
	br label %35

35:
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
	%1 = bitcast %ddpstring* %s1 to i8*
	call void @inbuilt_increment_ref_count(i8* %1, i8 0)
	%2 = alloca %ddpstring*
	%3 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %s1)
	store %ddpstring* %3, %ddpstring** %2
	%4 = bitcast %ddpstring* %s1 to i8*
	call void @inbuilt_decrement_ref_count(i8* %4)
	%5 = bitcast %ddpstring* %s2 to i8*
	call void @inbuilt_increment_ref_count(i8* %5, i8 0)
	%6 = alloca %ddpstring*
	%7 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %s2)
	store %ddpstring* %7, %ddpstring** %6
	%8 = bitcast %ddpstring* %s2 to i8*
	call void @inbuilt_decrement_ref_count(i8* %8)
	%9 = load %ddpstring*, %ddpstring** %2
	%10 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %9)
	call void @inbuilt_Schreibe_Text(%ddpstring* %10)
	call void @inbuilt_Schreibe_Buchstabe(i16 10)
	%11 = load %ddpstring*, %ddpstring** %6
	%12 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %11)
	call void @inbuilt_Schreibe_Text(%ddpstring* %12)
	call void @inbuilt_Schreibe_Buchstabe(i16 10)
	%13 = bitcast [7 x i16]* @0 to i16*
	%14 = call %ddpstring* @inbuilt_string_from_constant(i16* %13, i64 7)
	%15 = bitcast %ddpstring* %14 to i8*
	call void @inbuilt_increment_ref_count(i8* %15, i8 0)
	%16 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %14)
	%17 = bitcast %ddpstring* %14 to i8*
	call void @inbuilt_decrement_ref_count(i8* %17)
	%18 = load %ddpstring*, %ddpstring** %2
	%19 = bitcast %ddpstring* %18 to i8*
	call void @inbuilt_decrement_ref_count(i8* %19)
	%20 = load %ddpstring*, %ddpstring** %6
	%21 = bitcast %ddpstring* %20 to i8*
	call void @inbuilt_decrement_ref_count(i8* %21)
	ret %ddpstring* %16
}
