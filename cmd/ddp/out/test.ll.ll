; ModuleID = 'out/test.ll'
source_filename = "test.ddp"
target datalayout = "e-m:w-p270:32:32-p271:32:32-p272:64:64-i64:64-f80:128-n8:16:32:64-S128"
target triple = "x86_64-w64-windows-gnu"

%ddpstring = type { %garbage_collected, i16*, i64 }
%garbage_collected = type { %garbage_collected*, i1, i8 }

@0 = global [7 x i16] [i16 69, i16 114, i16 102, i16 111, i16 108, i16 103, i16 33]
@1 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 49]
@2 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 50]
@3 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 49]
@4 = global [10 x i16] [i16 78, i16 97, i16 99, i16 104, i16 114, i16 105, i16 99, i16 104, i16 116, i16 50]

declare %ddpstring* @inbuilt_string_from_constant(i16*, i64)

declare %ddpstring* @inbuilt_deep_copy_string(%ddpstring*)

declare void @mark_gc(%garbage_collected*)

define i64 @inbuilt_ddpmain() {
  %1 = alloca i64, align 8
  br label %2

2:                                                ; preds = %4, %0
  %storemerge = phi i64 [ 0, %0 ], [ %9, %4 ]
  store i64 %storemerge, i64* %1, align 8
  %3 = icmp eq i64 %storemerge, 10000
  br i1 %3, label %10, label %4

4:                                                ; preds = %2
  %5 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([10 x i16], [10 x i16]* @1, i64 0, i64 0), i64 10)
  %6 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([10 x i16], [10 x i16]* @2, i64 0, i64 0), i64 10)
  %7 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %5, %ddpstring* %6)
  %.cast4 = bitcast %ddpstring* %7 to %garbage_collected*
  call void @mark_gc(%garbage_collected* %.cast4)
  %.cast15 = bitcast %ddpstring* %7 to %garbage_collected*
  call void @mark_gc(%garbage_collected* %.cast15)
  %8 = load i64, i64* %1, align 8
  %9 = add i64 %8, 1
  br label %2

10:                                               ; preds = %2
  %11 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([10 x i16], [10 x i16]* @3, i64 0, i64 0), i64 10)
  %12 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([10 x i16], [10 x i16]* @4, i64 0, i64 0), i64 10)
  %13 = call %ddpstring* @ddpfunc_string_test(%ddpstring* %11, %ddpstring* %12)
  %.cast26 = bitcast %ddpstring* %13 to %garbage_collected*
  call void @mark_gc(%garbage_collected* %.cast26)
  %.cast37 = bitcast %ddpstring* %13 to %garbage_collected*
  call void @mark_gc(%garbage_collected* %.cast37)
  ret i64 0
}

declare void @inbuilt_Schreibe_Buchstabe(i16)

declare void @inbuilt_Schreibe_Text(%ddpstring*)

define %ddpstring* @ddpfunc_string_test(%ddpstring* %s1, %ddpstring* %s2) {
  %1 = alloca %ddpstring*, align 8
  %2 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %s1)
  store %ddpstring* %2, %ddpstring** %1, align 8
  %3 = bitcast %ddpstring* %s1 to %garbage_collected*
  call void @mark_gc(%garbage_collected* %3)
  %4 = alloca %ddpstring*, align 8
  %5 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %s2)
  store %ddpstring* %5, %ddpstring** %4, align 8
  %6 = bitcast %ddpstring* %s2 to %garbage_collected*
  call void @mark_gc(%garbage_collected* %6)
  %7 = load %ddpstring*, %ddpstring** %1, align 8
  call void @inbuilt_Schreibe_Text(%ddpstring* %7)
  call void @inbuilt_Schreibe_Buchstabe(i16 10)
  call void @inbuilt_Schreibe_Text(%ddpstring* %5)
  call void @inbuilt_Schreibe_Buchstabe(i16 10)
  %8 = call %ddpstring* @inbuilt_string_from_constant(i16* getelementptr inbounds ([7 x i16], [7 x i16]* @0, i64 0, i64 0), i64 7)
  %9 = call %ddpstring* @inbuilt_deep_copy_string(%ddpstring* %8)
  %10 = bitcast %ddpstring* %8 to %garbage_collected*
  call void @mark_gc(%garbage_collected* %10)
  %11 = bitcast %ddpstring** %1 to %garbage_collected**
  %12 = load %garbage_collected*, %garbage_collected** %11, align 8
  call void @mark_gc(%garbage_collected* %12)
  %13 = bitcast %ddpstring** %4 to %garbage_collected**
  %14 = load %garbage_collected*, %garbage_collected** %13, align 8
  call void @mark_gc(%garbage_collected* %14)
  ret %ddpstring* %9
}
